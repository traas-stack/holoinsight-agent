/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/cricore"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"io"
	"net"
	"time"
)

const (
	delayCancel = 500 * time.Millisecond
)

// TcpProxy
func TcpProxy(ctx context.Context, i cri.Interface, c *cri.Container, addr string, dialTimeout time.Duration) (net.Conn, error) {
	if c.Runtime == cri.Runc {
		sandbox := c.Pod.Sandbox
		if sandbox != nil {
			conn, err := cricore.NsEnterDial(ctx, c, "tcp", addr, dialTimeout)
			logger.Infozc(ctx, "[netproxy] runtime is runc, use nsenter", zap.Error(err))
			return conn, err
		}
	}
	return TcpProxyByExec(ctx, i, c, addr)
}

func TcpProxyByExec(ctx context.Context, i cri.Interface, c *cri.Container, addr string) (net.Conn, error) {
	pin, pout := io.Pipe()

	// Normally, the stream ends before exec ends.
	// If execCtx has been canceled at this time, exec will die from kill.
	// Although there is no actual loss (because the stream has been read) this will result in an error.
	// So we have to delay cancel execCtx
	execCtx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			time.AfterFunc(delayCancel, cancel)
		case <-execCtx.Done():
		}
	}()
	logger.Infozc(ctx, "[netproxy] use cri exec")
	ear, err := i.ExecAsync(execCtx, c, cri.ExecRequest{
		Cmd:    []string{core.HelperToolPath, "tcpProxy"},
		Env:    []string{"TCPPROXY_ADDR=" + addr, "TCPPROXY_IDLE_TIMEOUT=180s", "NO_JSON_OUTPUT=true"},
		Input:  pin,
		FixOut: true,
	})
	if err != nil {
		cancel()
		pout.CloseWithError(err)
		return nil, err
	}
	stderrCh := make(chan string, 1)
	go func() {
		bs, _ := io.ReadAll(ear.Stderr)
		stderrCh <- string(bs)
	}()

	go func() {
		defer cancel()
		var rc cri.ExecAsyncResultCode
		hasResult := false
		select {
		case <-execCtx.Done():
		case rc = <-ear.Result:
			hasResult = true
		}

		if !hasResult {
			rc = <-ear.Result
		}
		stderr := <-stderrCh
		logger.Infozc(ctx, "[netproxy] cri exec finished", zap.Int("code", rc.Code), zap.String("stderr", stderr), zap.Error(rc.Err))
		pout.CloseWithError(rc.Err)
	}()

	return &util.ReadWriterConn{
		Reader: ear.Stdout,
		Writer: pout,
		CloseFunc: func() {
			// If we cancel ctx immediately, then the bottom layer has a certain probability to return <-ctx.Done() instead of <-ear.Result, and there is competition here.
			//We prefer to leave the opportunity to <-ear.Result, so here is an appropriate delay of 100ms.
			// cancel happens before ear.Result
			time.AfterFunc(delayCancel, cancel)
		},
	}, nil
}
