/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import "github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"

func init() {
	model.RegisterHandler("hello", helloHandler)
	model.RegisterHandler("processes", processesHandler)
	model.RegisterHandler("glob", globHandler)
	model.RegisterHandler("listFiles", listFilesHandlers)
	model.RegisterHandler("httpcheck", httpCheckHandler)
	model.RegisterHandler("cat", catHandler)
	model.RegisterHandler("previewLog", previewLogHandler)
	model.RegisterHandler("inspect", inspectHandler)
	model.RegisterHandler("inputProxy", inputProxyHandler)
	model.RegisterHandler("httpProxy", httpProxyHandler)
	model.RegisterHandler("tcpProxy", tcpProxyHandler)
}
