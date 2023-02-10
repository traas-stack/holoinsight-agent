package handlers

import "github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"

func init() {
	model.RegisterHandler("hello", helloHandler)
	model.RegisterHandler("processes", processesHandler)
	model.RegisterHandler("glob", globHandler)
	model.RegisterHandler("listFiles", listFilesHandlers)
	model.RegisterHandler("httpcheck", httpCheckHandler)
	model.RegisterHandler("dialcheck", dialcheckHandler)
	model.RegisterHandler("cat", catHandler)
	model.RegisterHandler("previewLog", previewLogHandler)
	model.RegisterHandler("inspect", inspectHandler)
	model.RegisterHandler("collectInput", collectInputHandler)
}
