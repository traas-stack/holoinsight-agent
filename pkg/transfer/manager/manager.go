/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package manager

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/transfer"
	transferpb "github.com/traas-stack/holoinsight-agent/pkg/transfer/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

type (
	// TransferManager is responsible for handling the logic of lossless restart/deployment.
	TransferManager struct {
		pm             *pipeline.Manager
		lsm            *logstream.Manager
		stopComponents []StopComponent
	}
	StopComponent interface {
		Stop()
	}
)

var (
	// transferSockFile is usually /usr/local/holoinsight/agent/data/transfer.sock
	transferSockFile string
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	transferSockFile = filepath.Join(wd, "data", "transfer.sock")
}

func NewTransferManager(pm *pipeline.Manager, lsm *logstream.Manager) *TransferManager {
	return &TransferManager{
		pm:  pm,
		lsm: lsm,
	}
}

// StopAndSaveState stops all active components and dumps their state to []byte
func (tm *TransferManager) StopAndSaveState() ([]byte, error) {
	store := transfer.NewMemoryStateStore()
	begin := time.Now()
	pipelineSuccess := 0
	tm.pm.Update(func(pipelines map[string]api.Pipeline) {
		for _, p := range pipelines {
			sc, ok := p.(transfer.StatefulComponent)
			if !ok {
				continue
			}
			if err := sc.StopAndSaveState(store); err != nil {
				logger.Errorz("[transfer] [pipeline] save state error", zap.String("key", p.Key()), zap.Error(err))
			} else {
				pipelineSuccess++
				logger.Infoz("[transfer] [pipeline] save state success", zap.String("key", p.Key()))
			}
		}
	})

	{
		err := tm.lsm.StopAndSaveState(store)
		if err != nil {
			logger.Errorz("[transfer] [logstream] manager save state error", zap.Error(err))
			// If lsm fails to save state, returns err totally.
			return nil, err
		} else {
			logger.Infoz("[transfer] [logstream] manager save state success")
		}
	}

	stateBytes, err := util.GobEncode(store.State)
	if err != nil {
		return nil, err
	}

	cost := time.Now().Sub(begin)

	logger.Infoz("[transfer] [server] stop and save state success",
		zap.Int("bytes", len(stateBytes)),
		zap.Duration("cost", cost),
	)

	return stateBytes, nil
}

func (tm *TransferManager) TransferDone() {
	logger.Infoz("[transfer] [server] TransferDone, agent will exit with code=7 in 1s")

	time.AfterFunc(1*time.Second, func() {
		os.Exit(7)
	})
}

func (tm *TransferManager) callPauseAndSaveState(client transferpb.TransferSrviceClient) (*transferpb.StopAndSaveSaveResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return client.StopAndSaveState(ctx, &transferpb.StopAndSaveSaveRequest{})
}

func (tm *TransferManager) callTransferDone(client transferpb.TransferSrviceClient) error {
	begin := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := client.TransferDone(ctx, &transferpb.TransferDoneRequest{})
	cost := time.Now().Sub(begin)
	logger.Infoz("[transfer] [client] TransferDone", zap.Any("resp", resp), zap.Duration("cost", cost), zap.Error(err))
	return err
}

func (tm *TransferManager) getRemoteState(client transferpb.TransferSrviceClient) (*transfer.MemoryStateStore, error) {
	begin := time.Now()
	resp, err := tm.callPauseAndSaveState(client)
	cost := time.Now().Sub(begin)
	if err != nil {
		logger.Infoz("[transfer] [client] PauseAndSaveSave error", zap.Duration("cost", cost), zap.Error(err))
		return nil, err
	}

	stateStore := transfer.NewMemoryStateStore()
	if err := util.GobDecode(resp.State, &stateStore.State); err != nil {
		logger.Errorz("[transfer] [client] PauseAndSaveSave decode error", zap.Duration("cost", cost), zap.Error(err))
		return nil, err
	}

	logger.Infoz("[transfer] [client] get remote state success", zap.Any("bytes", len(resp.State)), zap.Duration("cost", cost))

	return stateStore, nil
}

func (tm *TransferManager) loadState(stateStore *transfer.MemoryStateStore) error {
	if err := tm.lsm.LoadState(stateStore); err != nil {
		logger.Errorz("[transfer] [client] LogStream.Manager load state error", zap.Error(err))
		return err
	}

	logger.Infoz("[transfer] [client] LogStream.Manager load state success")

	pipelineCount := 0
	pipelineNoImpl := 0
	pipelineSuccess := 0
	pipelineError := 0
	tm.pm.Update(func(pipelines map[string]api.Pipeline) {
		// Currently, state dump is quick. So no need to be concurrent here.
		for _, p := range pipelines {
			pipelineCount++
			if sc, ok := p.(transfer.StatefulComponent); ok {
				if err := sc.LoadState(stateStore); err != nil {
					pipelineError++
					logger.Errorz("[transfer] [pipeline] load state error", zap.String("key", p.Key()), zap.Error(err))
				} else {
					pipelineSuccess++
					logger.Infoz("[transfer] [pipeline] load state success", zap.String("key", p.Key()))
				}
			} else {
				pipelineNoImpl++
			}
		}
	})
	logger.Infoz("[transfer] [client] pipelines load state",
		zap.Int("total", pipelineCount),
		zap.Int("success", pipelineSuccess),
		zap.Int("error", pipelineError),
		zap.Int("noImpl", pipelineNoImpl),
	)

	tm.lsm.CleanInvalidRefAfterLoadState()

	return nil
}

func (tm *TransferManager) transfer(conn *grpc.ClientConn) error {
	client := transferpb.NewTransferSrviceClient(conn)

	// call TransferDone whatever
	defer tm.callTransferDone(client)

	stateStore, err := tm.getRemoteState(client)
	if err != nil {
		return err
	}

	return tm.loadState(stateStore)
}

// createConn create grp connection to old instance
func (tm *TransferManager) createConn() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "", //
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(100*1024*1024)),
		grpc.WithTransportCredentials(insecure.NewCredentials()), //
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { //
			return net.DialTimeout("unix", transferSockFile, 3*time.Second)
		}))

	if err != nil {
		logger.Infoz("[transfer] [client] fail to connect to transfer.sock", zap.String("sock", transferSockFile), zap.Error(err))
		return nil, err
	}
	return conn, nil
}

// Transfer transfers state from old instance to current instance
func (tm *TransferManager) Transfer() error {
	defer os.Remove(transferSockFile)

	logger.Infoz("[transfer] [client] maybeTransfer begin")
	begin := time.Now()

	conn, err := tm.createConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	err = tm.transfer(conn)

	cost := time.Now().Sub(begin)
	logger.Infoz("[transfer] [client] transfer done", zap.Duration("cost", cost), zap.Error(err))

	return err

}

func (tm *TransferManager) ListenTransfer() {
	ls, err := net.Listen("unix", transferSockFile)
	if err != nil {
		logger.Errorz("[transfer] listen transfer.sock error",
			zap.String("sock", transferSockFile),
			zap.Error(err))
		return
	}

	defer ls.Close()
	logger.Infoz("[transfer] listen transfer.sock success",
		zap.String("sock", transferSockFile))

	grpcServer := grpc.NewServer()
	transferpb.RegisterTransferSrviceServer(grpcServer, &transferSrviceServerImpl{
		tm: tm,
	})

	if err := grpcServer.Serve(ls); err != nil {
		logger.Errorz("[transfer] grpc serve error", zap.Error(err))
	}
}

func (tm *TransferManager) AddStopComponents(stopComponents ...StopComponent) {
	tm.stopComponents = append(tm.stopComponents, stopComponents...)
}

func (tm *TransferManager) prepare() {
	logger.DisableRotates()
	logger.Infoz("[transfer] [server] disable log rotates")
	for _, component := range tm.stopComponents {
		begin := time.Now()
		component.Stop()
		cost := time.Now().Sub(begin)
		logger.Infoz("[transfer] [server] stop component", zap.Any("component", reflect.TypeOf(component)), zap.Duration("cost", cost))
	}
}
