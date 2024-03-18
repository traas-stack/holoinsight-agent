/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package netproxy

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"net"
)

type (
	PortForwardTask struct {
		Addr string
	}
)

func (t *PortForwardTask) Start(ctx context.Context) (string, error) {
	i := ioc.Crii
	host, _, err := net.SplitHostPort(t.Addr)
	if err != nil {
		logger.Errorz("[netproxy] parse host error", zap.String("addr", t.Addr), zap.Error(err))
		return "", err
	}

	pod := criutils.FindFirstPodByIp(i, host)
	if pod == nil {
		return "", errors.New("no pod")
	}
	biz := pod.Biz[0]

	// ':0' uses random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	stopCh := make(chan struct{}, 2)
	defer func() {
		stopCh <- struct{}{}
	}()

	// listen to ctx Done
	go func() {
		select {
		case <-stopCh:
		case <-ctx.Done():
			listener.Close()
			stopCh <- struct{}{}
		}
	}()
	logCtx := zap.Fields(zap.String("uuid", uuid.New().String()), zap.String("cid", biz.ShortContainerID()), zap.String("listenAddr", listener.Addr().String()), zap.String("toAddr", t.Addr))
	logger.Infozo(logCtx, "[netproxy] create port forward")

	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				logger.Errorzo(logCtx, "[netproxy] portforward listen error", zap.Error(err))
				return
			}
			go func() {
				handlePortForwardRequest(logCtx, biz, conn, t.Addr)
			}()
		}
	}()

	return listener.Addr().String(), nil
}

func handlePortForwardRequest(logCtx zap.Option, biz *cri.Container, conn net.Conn, addr string) {
	defer conn.Close()

	subConn, err := TcpProxy(logger.WithLogCtx(context.Background(), logCtx), ioc.Crii, biz, addr, DefaultDialTimeout)
	if err != nil {
		panic(err)
	}

	defer subConn.Close()

	err = util.CopyConn(logger.WithLogCtx(context.Background(), logCtx), conn, subConn, DefaultSocketTimeout)
	if err != nil {
		logger.Errorzo(logCtx, "[netproxy] portforward conn error", zap.Error(err))
	}
}
