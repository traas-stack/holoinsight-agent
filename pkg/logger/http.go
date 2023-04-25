/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logger

import (
	"net/http"
	"time"
)

func registerHttpHandler() {
	http.HandleFunc("/api/log/debug/start", func(writer http.ResponseWriter, request *http.Request) {
		DebugEnabled = true
		writer.Write([]byte("OK"))
	})
	http.HandleFunc("/api/log/debug/stop", func(writer http.ResponseWriter, request *http.Request) {
		DebugEnabled = false
		writer.Write([]byte("OK"))
	})
	go func() {
		// 自动关闭
		timer := time.NewTimer(10 * time.Hour)
		defer timer.Stop()
		for range timer.C {
			DebugEnabled = false
		}
	}()
}
