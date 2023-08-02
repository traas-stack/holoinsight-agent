package handlers

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"io"
	"net"
	"os"
	"time"
)

const (
	defaultDialTimeout = 2 * time.Second
)

func tcpProxyHandler(_ string, _ *model.Resp) error {
	addr := os.Getenv("TCPPROXY_ADDR")
	// If conn has no traffic within 1 minute, it will cause an error
	timeout := util.ParseDurationDefault(os.Getenv("TCPPROXY_TIMEOUT"), time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return tcpProxyHandler0(ctx, addr, timeout, os.Stdin, os.Stdout)
}

func tcpProxyHandler0(ctx context.Context, addr string, timeout time.Duration, in io.Reader, out io.Writer) error {
	conn, err := net.DialTimeout("tcp", addr, defaultDialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	return util.CopyConn(ctx, conn, &util.ReadWriterConn{
		Reader: in,
		Writer: out,
	}, timeout)
}
