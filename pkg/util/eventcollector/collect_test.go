package eventcollector

import (
	"testing"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

func TestEventCollector(t *testing.T) {
	appconfig.SetDev(true)
	logger.SetupZapLogger()

	ec := New(time.Second, func(events []*Event) {
		for _, v := range events {
			t.Log(v)
		}
	})

	ec.Collect("cfgInitError", "slice index out of range!")

	time.Sleep(time.Second * 5)
}
