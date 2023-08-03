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

// TcpProxy
func TcpProxy(ctx context.Context, i cri.Interface, c *cri.Container, addr string, dialTimeout time.Duration) (net.Conn, error) {
	if c.Runtime == cri.Runc {
		sandbox := c.Pod.Sandbox
		if sandbox != nil {
			return cricore.NsEnterDial(c, "tcp", addr, dialTimeout)
		}
	}

	pin, pout := io.Pipe()
	ctx2, cancel := context.WithCancel(ctx)
	ear, err := i.ExecAsync(ctx2, c, cri.ExecRequest{
		Cmd:   []string{core.HelperToolPath, "tcpProxy"},
		Env:   []string{"TCPPROXY_ADDR=" + addr, "TCPPROXY_IDLE_TIMEOUT=60s", "NO_JSON_OUTPUT=true"},
		Input: pin,
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
		case <-ctx.Done():
		case rc = <-ear.Result:
			hasResult = true
		}
		if !hasResult {
			rc = <-ear.Result
		}
		stderr := <-stderrCh
		logger.Infozc(ctx, "[netproxy] tcpproxy exec finished", zap.Int("code", rc.Code), zap.String("stderr", stderr), zap.Error(rc.Err))
		pout.CloseWithError(rc.Err)
	}()

	return &util.ReadWriterConn{
		Reader: ear.Stdout,
		Writer: pout,
		CloseFunc: func() {
			// If we cancel ctx immediately, then the bottom layer has a certain probability to return <-ctx.Done() instead of <-ear.Result, and there is competition here.
			//We prefer to leave the opportunity to <-ear.Result, so here is an appropriate delay of 100ms.
			time.AfterFunc(100*time.Millisecond, cancel)
		},
	}, nil
}
