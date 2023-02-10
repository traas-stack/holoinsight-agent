package server

import (
	"net/http"
	"testing"

	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

func TestHttpServer(t *testing.T) {
	appconfig.SetDev(true)
	logger.SetupZapLogger()
	StartHTTPController()

	RegisterApiHandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		t.Logf("test url")
		writer.Write([]byte("test success!"))
	})

	<-make(chan struct{})
}
