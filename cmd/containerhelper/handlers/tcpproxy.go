/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const (
	defaultDialTimeout = 2 * time.Second
)

func tcpProxyHandler(_ string, _ *model.Resp) error {
	addr := os.Getenv("TCPPROXY_ADDR")
	// If conn has no traffic within 1 minute, it will cause an error
	idleTimeout := util.ParseDurationDefault(os.Getenv("TCPPROXY_IDLE_TIMEOUT"), time.Minute)
	ctx := context.Background()
	return tcpProxyHandler0(ctx, addr, idleTimeout, os.Stdin, os.Stdout)
}

func tcpProxyHandler0(ctx context.Context, addr string, idleTimeout time.Duration, in io.Reader, out io.Writer) error {
	var conn net.Conn
	var err error
	if strings.HasSuffix(addr, ".sock") {
		conn, err = net.DialTimeout("unix", addr, defaultDialTimeout)
	} else {
		conn, err = net.DialTimeout("tcp", addr, defaultDialTimeout)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	return util.CopyConn(ctx, conn, &util.ReadWriterConn{
		Reader: in,
		Writer: out,
	}, idleTimeout)
}
