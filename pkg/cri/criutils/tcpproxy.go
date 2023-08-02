package criutils

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/cricore"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
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
		Env:   []string{"TCPPROXY_ADDR=" + addr, "TCPPROXY_TIMEOUT=60s", "NO_JSON_OUTPUT=true"},
		Input: pin,
	})
	if err != nil {
		return nil, err
	}

	return &util.ReadWriterConn{
		Reader: ear.Stdout,
		Writer: pout,
		Cancel: cancel,
	}, nil
}
