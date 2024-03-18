/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package netproxy

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"github.com/txthinking/socks5"
	"go.uber.org/zap"
	"io"
	net2 "k8s.io/apimachinery/pkg/util/net"
	"net"
	"net/url"
	"sync/atomic"
	"time"
)

const (
	Socks5ProxyAddr = "127.0.0.1:9119"
)

var (
	Socks5ProxyURL, _  = url.Parse(fmt.Sprintf("socks5://%s", Socks5ProxyAddr))
	netproxyCreateStat = stat.DefaultManager.Counter("netproxy.create")
)

type (
	// CriHandle implements socks5.Handler interface
	CriHandle struct {
		Cri cri.Interface
		wip int32
	}
)

func (h *CriHandle) Init() {
	stat.DefaultManager.Gauge("netproxy.holding", func() []stat.GaugeSubItem {
		return []stat.GaugeSubItem{
			{
				Keys:   []string{"socks5"},
				Values: []int64{int64(atomic.LoadInt32(&h.wip))},
			},
		}
	})
}

// TCPHandle handle tcp socks5 proxy request.
func (h *CriHandle) TCPHandle(s *socks5.Server, c *net.TCPConn, r *socks5.Request) error {
	netproxyCreateStat.Add([]string{"socks5"}, stat.V_1)
	atomic.AddInt32(&h.wip, 1)
	defer func() {
		atomic.AddInt32(&h.wip, -1)
	}()
	h.tcpHandle(s, c, r)
	return nil
}

func (h *CriHandle) tcpHandle(s *socks5.Server, c *net.TCPConn, r *socks5.Request) error {
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
			a, addr, port, _ := socks5.ParseAddress(Socks5ProxyAddr)
			rep := socks5.RepHostUnreachable
			socks5.NewReply(rep, a, addr, port).WriteTo(c)
			logger.Errorz("no pod when proxy", zap.String("ip", host))
			return errors.New("no pod")
		}
		if len(pod.Biz) == 0 {
			return errors.New("biz is nil")
		}
		// TODO which one?
		biz = pod.Biz[0]
	}

	uuid2 := uuid.New().String()

	logCtx := zap.Fields(zap.String("uuid", uuid2), zap.String("protocol", "socks5"), zap.String("cid", biz.ShortContainerID()), zap.String("addr", addr))

	proxied, err := TcpProxy(logger.WithLogCtx(context.Background(), logCtx), h.Cri, biz, addr, DefaultDialTimeout)
	if err != nil {
		logger.Infozo(logCtx, "[netproxy] create tcperror error", zap.Error(err))
		a, addr, port, _ := socks5.ParseAddress(Socks5ProxyAddr)
		rep := socks5.RepServerFailure
		if net2.IsConnectionRefused(err) {
			rep = socks5.RepConnectionRefused
		}
		p := socks5.NewReply(rep, a, addr, port)
		if _, err := p.WriteTo(c); err != nil {
			return err
		}
		return err
	}

	// handshake
	{
		a, addr, port, _ := socks5.ParseAddress(Socks5ProxyAddr)
		p := socks5.NewReply(socks5.RepSuccess, a, addr, port)
		if _, err := p.WriteTo(c); err != nil {
			return err
		}
	}
	logger.Infozo(logCtx, "[netproxy] stream created")
	defer proxied.Close()

	// copy streams
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(proxied, &util.ReaderReadHook{Reader: c, Before: func() {
			if s.TCPTimeout > 0 {
				c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second))
			}
		}})
		errCh <- err
		util.MaybeCloseWrite(proxied)
		c.CloseRead()
		io.Copy(io.Discard, c)
	}()
	go func() {
		_, err := io.Copy(c, &util.ReaderReadHook{Reader: proxied, Before: func() {
			if s.TCPTimeout > 0 {
				proxied.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second))
			}
		}})
		errCh <- err
		c.CloseWrite()
		util.MaybeCloseRead(proxied)
		io.Copy(io.Discard, proxied)
	}()

	wait := 2
	for {
		select {
		case err := <-errCh:
			if err != nil && err != io.EOF {
				logger.Errorzo(logCtx, "[netproxy] stream error", zap.Error(err))
				return err
			}
			wait--
			if wait == 0 {
				logger.Infozo(logCtx, "[netproxy] stream finished")
				return nil
			}
		}
	}

}

// UDPHandle auto handle packet. You may prefer to do yourself.
func (h *CriHandle) UDPHandle(s *socks5.Server, addr *net.UDPAddr, d *socks5.Datagram) error {
	return errors.New("unsupported")
}

func TcpProxy(ctx context.Context, i cri.Interface, c *cri.Container, addr string, dialTimeout time.Duration) (net.Conn, error) {
	for _, handler := range tcpHandlers {
		if conn, err := handler(ctx, i, c, addr, dialTimeout); conn != nil && err == nil {
			return conn, err
		}
	}
	return criutils.TcpProxy(ctx, i, c, addr, dialTimeout)
}
