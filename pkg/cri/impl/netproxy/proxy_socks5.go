package netproxy

import (
	"context"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/txthinking/socks5"
	"go.uber.org/zap"
	"io"
	"net"
	"net/url"
	"time"
)

const (
	Socks5ProxyAddr = "127.0.0.1:9119"
)

var (
	Socks5ProxyURL, _ = url.Parse(fmt.Sprintf("socks5://%s", Socks5ProxyAddr))
)

type (
	// CriHandle implements socks5.Handler interface
	CriHandle struct {
		Cri cri.Interface
	}
)

// TCPHandle handle tcp socks5 proxy request.
func (h *CriHandle) TCPHandle(s *socks5.Server, c *net.TCPConn, r *socks5.Request) error {
	if r.Cmd != socks5.CmdConnect {
		return socks5.ErrUnsupportCmd
	}
	if !NETPROXY_ENABLED {
		return (&socks5.DefaultHandle{}).TCPHandle(s, c, r)
	}

	var biz *cri.Container
	addr := r.Address()
	{
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}
		pod := criutils.FindFirstPodByIp(h.Cri, host)
		if pod == nil {
			return errors.New("no pod")
		}
		if len(pod.Biz) == 0 {
			return errors.New("biz is nil")
		}
		// TODO which one?
		biz = pod.Biz[0]
	}

	logCtx := zap.Fields(zap.String("protocol", "socks5"), zap.String("cid", biz.ShortContainerID()), zap.String("addr", addr))

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	logger.ZapLogger.Info.WithOptions(logCtx).Info("[netproxy] create")
	proxied, err := criutils.TcpProxy(ctx, h.Cri, biz, addr, DefaultDialTimeout)
	if err != nil {
		logger.Infoz("[netproxy] create tcperror error", zap.Error(err))
		return err
	}
	defer proxied.Close()

	// handshake
	{
		a, addr, port, _ := socks5.ParseAddress(Socks5ProxyAddr)
		p := socks5.NewReply(socks5.RepSuccess, a, addr, port)
		if _, err := p.WriteTo(c); err != nil {
			return err
		}
	}

	// copy streams
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(proxied, &util.ReaderReadHook{Reader: c, Before: func() {
			if s.TCPTimeout > 0 {
				c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second))
			}
		}})
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(c, &util.ReaderReadHook{Reader: proxied, Before: func() {
			proxied.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second))
		}})
		errCh <- err
	}()

	wait := 2
	for {
		select {
		case err := <-errCh:
			if err != nil {
				logger.ZapLogger.Error.WithOptions(logCtx).Info("[netproxy] stream error")
				return err
			}
			wait--
			if wait == 0 {
				logger.ZapLogger.Error.WithOptions(logCtx).Info("[netproxy] stream finished")
				return nil
			}
		}
	}

}

// UDPHandle auto handle packet. You may prefer to do yourself.
func (h *CriHandle) UDPHandle(s *socks5.Server, addr *net.UDPAddr, d *socks5.Datagram) error {
	return errors.New("unsupported")
}
