/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package server

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/util/recoverutils"
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	_ "net/http/pprof"
)

var bindPort = 9117

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

// 启动http服务
func StartHTTPController() {
	apiHandleFuncMux.HandleFunc("/", printHelp)

	var f func()
	f = func() {
		recoverutils.GoWithRecover(func() {
			addr := fmt.Sprintf("127.0.0.1:%d", bindPort)
			logger.Infoz("[debug] start debug http server", zap.String("addr", addr))
			err := http.ListenAndServe(addr, apiHandleFuncMux)
			if err != nil {
				logger.Errorz("listen and serve debug http server fail", zap.Error(err))
				return
			}
		}, func(_ interface{}) { f() })
	}
	f()
}
