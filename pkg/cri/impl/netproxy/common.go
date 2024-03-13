/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package netproxy

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"net"
	"net/http"
	"time"
)

const (
	DefaultDialTimeout   = 2 * time.Second
	DefaultSocketTimeout = 3 * time.Minute
)

type (
	HttpHandler func(ctx context.Context, pod *cri.Pod, req *http.Request) (*http.Request, *http.Response, error)
	TcpHandler  func(ctx context.Context, i cri.Interface, c *cri.Container, addr string, dialTimeout time.Duration) (net.Conn, error)
)

var (
	handlers    []HttpHandler
	tcpHandlers []TcpHandler
)

func AddHttpProxyHandler(handler HttpHandler) {
	handlers = append(handlers, handler)
}

func AddTcpProxyHandler(handler TcpHandler) {
	tcpHandlers = append(tcpHandlers, handler)
}
