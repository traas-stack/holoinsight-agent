/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package server

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	_ "net/http/pprof"
)

const (
	bindPort = 9117
)

type (
	HttpServerComponent struct {
		server *http.Server
		mutex  sync.Mutex
	}
)

func NewHttpServerComponent() *HttpServerComponent {
	return &HttpServerComponent{}
}

func (h *HttpServerComponent) Start() {
	go func() {
		apiHandleFuncMux.HandleFunc("/", printHelp)

		bindIp := "127.0.0.1"
		if appconfig.StdAgentConfig.Mode != core.AgentModeDaemonset {
			bindIp = "0.0.0.0"
		}
		addr := fmt.Sprintf("%s:%d", bindIp, bindPort)

		logger.Infoz("[http] start http server", zap.String("addr", addr))
		h.server = &http.Server{Addr: addr, Handler: apiHandleFuncMux}
		if err := h.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				logger.Errorz("[http] server closed", zap.String("addr", addr))
			} else {
				logger.Errorz("[http] listen and serve error", zap.String("addr", addr), zap.Error(err))
			}
			return
		}

	}()
}

func (h *HttpServerComponent) Stop() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	server := h.server
	if server != nil {
		h.server = nil
		server.Close()
	}
}

func buildHelps() string {
	apiHandleFuncStoreMu.RLock()
	m := apiHandleFuncStore
	apiHandleFuncStoreMu.RUnlock()

	helps := "Some help msg for monitor-agent:\n"
	const blankNum = 25

	var urls []string
	for k := range m {
		urls = append(urls, k)
	}
	sort.Strings(urls)

	for _, k := range urls {
		v := m[k]
		name := strings.Split(runtime.FuncForPC(reflect.ValueOf(v.F).Pointer()).Name(), ".")
		hName := name[len(name)-1] + ":"
		usage := fmt.Sprintf("curl 127.0.0.1:%d%s", bindPort, k)
		for _, vv := range v.MoreUsages {
			if vv != "" {
				blank := ""
				for j := 0; j < blankNum; j++ {
					blank += " "
				}
				usage += fmt.Sprintf("\n%s", blank+vv)
			}
		}

		helps += fmt.Sprintf("%-25s%s\n", hName, usage)
	}
	return helps
}

func printHelp(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte(buildHelps()))
}
