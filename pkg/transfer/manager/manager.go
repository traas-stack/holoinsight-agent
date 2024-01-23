/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package manager

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/spf13/cast"
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

const (
	stateFileFreshDuration = 2 * time.Minute
	storeVersion           = "2"
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
	stateStoreStateObj struct {
		BornTime time.Time
		State    []byte
	}
)

var (
	// transferSockFile is usually /usr/local/holoinsight/agent/data/transfer.sock
	transferSockFile string
	stateFile        string
)

func init() {
	gob.Register(&stateStoreStateObj{})
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	transferSockFile = filepath.Join(wd, "data", "transfer.sock")
	stateFile = filepath.Join(wd, "data", "state")
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

	store.Put("storeVersion", storeVersion)

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
		// 0 will be treated as SUCCESS by k8s
		// Used with 'restartPolicy: OnFailure' or 'supervisord'
		os.Exit(0)
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
	logger.Infoz("[transfer] [client] transfer done", zap.Any("resp", resp), zap.Duration("cost", cost), zap.Error(err))
	return err
}

func (tm *TransferManager) getRemoteState(client transferpb.TransferSrviceClient) (*transfer.MemoryStateStore, error) {
	begin := time.Now()
	resp, err := tm.callPauseAndSaveState(client)
	cost := time.Now().Sub(begin)
	if err != nil {
		logger.Infoz("[transfer] [client] get remote state error", zap.Duration("cost", cost), zap.Error(err))
		return nil, err
	}

	stateStore := transfer.NewMemoryStateStore()
	if err := util.GobDecode(resp.State, &stateStore.State); err != nil {
		return nil, err
	}

	logger.Infoz("[transfer] [client] get remote state success", zap.Any("bytes", len(resp.State)), zap.Duration("cost", cost))

	return stateStore, nil
}

func (tm *TransferManager) loadState(stateStore *transfer.MemoryStateStore) error {
	{
		// version check
		stateVersion, err := stateStore.Get("storeVersion")
		if err != nil {
			return err
		}
		if storeVersion != cast.ToString(stateVersion) {
			return fmt.Errorf("old state has different version, version=%s epected=%s", cast.ToString(stateVersion), storeVersion)
		}
	}

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
func (tm *TransferManager) transferUsingGrpc() error {
	// transfer using grpc
	conn, err := tm.createConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := transferpb.NewTransferSrviceClient(conn)

	// call TransferDone whatever
	defer tm.callTransferDone(client)

	stateStore, err := tm.getRemoteState(client)
	if err != nil {
		return err
	}

	return tm.loadState(stateStore)
}

func (tm *TransferManager) transferUsingStateFile() error {
	logger.Infoz("[transfer] try state file", zap.String("file", stateFile))
	b, err := os.ReadFile(stateFile)
	if err != nil {
		logger.Infoz("[transfer] read state file error", zap.Error(err))
		return err
	}
	logger.Infoz("[transfer] load state from state file", zap.Int("bytes", len(b)))
	stateObj := &stateStoreStateObj{}
	if err := util.GobDecode(b, stateObj); err != nil {
		return err
	}
	logger.Infoz("[transfer] load state success", zap.Time("stateTime", stateObj.BornTime))
	now := time.Now()
	if stateObj.BornTime.Add(stateFileFreshDuration).Before(now) {
		return fmt.Errorf("state file is stale, state time=[%s] now=[%s]", stateObj.BornTime.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	stateStore := transfer.NewMemoryStateStore()
	if err := util.GobDecode(stateObj.State, &stateStore.State); err != nil {
		return err
	}
	return tm.loadState(stateStore)
}

// Transfer transfers state from old instance to current instance
func (tm *TransferManager) Transfer() error {
	defer func() {
		os.Remove(transferSockFile)
		os.Remove(stateFile)
	}()

	logger.Infoz("[transfer] [client] transfer begin")
	begin := time.Now()

	err := tm.transferUsingGrpc()

	if err != nil {
		err2 := tm.transferUsingStateFile()
		if !os.IsNotExist(err2) {
			err = err2
		}
	}

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

func (tm *TransferManager) StopSaveStateToFile() error {
	tm.prepare()
	state, err := tm.StopAndSaveState()
	if err != nil {
		return err
	}
	stateObj := &stateStoreStateObj{
		BornTime: time.Now(),
		State:    state,
	}
	b, err := util.GobEncode(stateObj)
	if err != nil {
		return err
	}
	logger.Infoz("[transfer] [server] save state to file", zap.String("file", stateFile))
	return os.WriteFile(stateFile, b, 0644)
}
