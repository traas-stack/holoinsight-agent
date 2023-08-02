package impl

import (
	"context"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/impl/netproxy"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/txthinking/socks5"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

const (
	portForwardSurvivalTime = 10 * time.Minute
)

func (e *defaultCri) startHttpProxyServer() {
	httpProxyServer := goproxy.NewProxyHttpServer()
	httpProxyServer.OnRequest().DoFunc(netproxy.HandleHttpProxy)
	e.httpProxyServer = &http.Server{Addr: netproxy.HttpProxyAddr, Handler: httpProxyServer}

	logCtx := zap.Fields(zap.String("protocol", "http"), zap.String("addr", netproxy.HttpProxyAddr))

	logger.Infozo(logCtx, "[netproxy] listen")

	go func() {
		err := e.httpProxyServer.ListenAndServe()
		if err == http.ErrServerClosed {
			logger.Infozo(logCtx, "[netproxy] closed")
		} else {
			logger.Errorzo(logCtx, "[netproxy] ListenAndServe error", zap.Error(err))
		}
	}()
}

func (e *defaultCri) startSocks5ProxyServer() {
	server, err := socks5.NewClassicServer(netproxy.Socks5ProxyAddr, "127.0.0.1", "", "", int(netproxy.DefaultSocketTimeout.Seconds()), 0)
	if err != nil {
		panic(err)
	}

	handler := &netproxy.CriHandle{
		Cri: e,
	}

	logCtx := zap.Fields(zap.String("protocol", "socks5"), zap.String("addr", netproxy.Socks5ProxyAddr))

	logger.Infozo(logCtx, "[netproxy] listen")
	go func() {
		err := server.ListenAndServe(handler)
		if strings.Contains(err.Error(), "use of closed network connection") {
			logger.Infozo(logCtx, "[netproxy] closed", zap.Error(err))
		} else {
			logger.Errorzo(logCtx, "[netproxy] ListenAndServe error", zap.Error(err))
		}
	}()

	e.socks5ProxyServer = server
}

func (e *defaultCri) listenPortForward() {
	http.HandleFunc("/api/cri/netproxy/portforward", func(writer http.ResponseWriter, request *http.Request) {
		addr := request.URL.Query().Get("addr")

		t := &netproxy.PortForwardTask{
			Addr: addr,
		}

		ctx, _ := context.WithTimeout(context.Background(), portForwardSurvivalTime)
		listenAddr, err := t.Start(ctx)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte(err.Error()))
		} else {
			writer.WriteHeader(http.StatusOK)
			text := fmt.Sprintf("Listen on %s, forwarding to %s\nThis connection will be closed in %s.", listenAddr, addr, portForwardSurvivalTime.String())
			writer.Write([]byte(text))
		}
	})
}
