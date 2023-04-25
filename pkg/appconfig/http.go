/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package appconfig

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

var uptime = time.Now()

func init() {
	http.HandleFunc("/version", func(writer http.ResponseWriter, request *http.Request) {
		r := map[string]interface{}{
			"goversion": runtime.Version(),
			"version":   agentVersion,
			"buildTime": agentBuildTime,
			"commit":    gitcommit,
			"uptime":    uptime.Format(time.RFC3339),
		}
		json.NewEncoder(writer).Encode(r)
	})
}
