/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"context"
	"github.com/rs/dnscache"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type (
	DnsCacheHelper struct {
		resolver    *dnscache.Resolver
		nextIpIndex int64
		stopCh      chan struct{}
		stopOnce    sync.Once
	}
)

var (
	localIp  string
	hostname string
	timezone string
)

func init() {
	{
		// TODO 多网卡场景
		// eth0
		firstIp := ""
		eth0Ip := ""

		interfaces, err := net.Interfaces()
		if err != nil {
			panic(err)
		}
	interfaceLoop:
		for _, i := range interfaces {
			if i.Name == "lo" {
				continue
			}
			if i.Flags&net.FlagUp != net.FlagUp {
				continue
			}
			if i.Flags&net.FlagLoopback == net.FlagLoopback {
				continue
			}
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				ipv4 := ipNet.IP.To4()
				if ipv4 == nil {
					continue
				}
				ip := ipv4.String()
				if firstIp == "" {
					firstIp = ip
				}
				if i.Name == "eth0" {
					eth0Ip = ip
					break interfaceLoop
				}
			}
		}

		if eth0Ip != "" {
			localIp = eth0Ip
		} else {
			localIp = firstIp
		}
	}

	{
		if h, err := os.Hostname(); err == nil && h != "" {
			hostname = h
		} else {
			hostname = localIp
		}
	}
}

// GetLocalIp get local ipv4
func GetLocalIp() string {
	return localIp
}

// GetHostname
func GetHostname() string {
	return hostname
}

func SetLocalIp(ip string) {
	localIp = ip
}

func GetLocalTimezone() string {
	tz, _ := time.Now().Zone()
	return tz
}

type (
	closeWrite interface {
		CloseWrite() error
	}
	closeRead interface {
		CloseRead() error
	}
)

func MaybeCloseWrite(conn interface{}) error {
	if x, ok := conn.(closeWrite); ok {
		return x.CloseWrite()
	}
	return nil
}

func MaybeCloseRead(conn interface{}) error {
	if x, ok := conn.(closeRead); ok {
		return x.CloseRead()
	}
	return nil
}

func NewDnsCacheHelper() *DnsCacheHelper {
	options := dnscache.ResolverRefreshOptions{}
	options.ClearUnused = true
	options.PersistOnFailure = false

	h := &DnsCacheHelper{
		resolver: &dnscache.Resolver{},
		stopCh:   make(chan struct{}),
	}

	h.resolver.RefreshWithOptions(options)
	return h
}

func (h *DnsCacheHelper) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-h.stopCh:
				return
			case <-ticker.C:
				h.resolver.Refresh(true)
			}
		}
	}()
}

func (h *DnsCacheHelper) Stop() {
	h.stopOnce.Do(func() { close(h.stopCh) })
}

func (h *DnsCacheHelper) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := h.resolver.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}

	size := len(ips)
	var lastErr error
	for i := 0; i < size; i++ {
		index := atomic.AddInt64(&h.nextIpIndex, 1)
		ip := ips[int(index)%size]

		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		if err == nil {
			return conn, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func (h *DnsCacheHelper) NewHttpClient() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: h.Dial}}
}

func dial(ctx context.Context, network, addr string) (net.Conn, error) {
	return NewDnsCacheHelper().Dial(ctx, network, addr)
}

func ReplaceHost(hostport string, host string) string {
	_, port, err := net.SplitHostPort(hostport)
	if err == nil {
		return net.JoinHostPort(host, port)
	}
	return host
}

func ReplaceHostToLocalhost(hostport string) string {
	return ReplaceHost(hostport, "127.0.0.1")
}
