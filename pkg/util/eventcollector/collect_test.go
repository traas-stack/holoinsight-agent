package eventcollector

import (
	"testing"
	"time"

	"github.com/TRaaSStack/holoinsight-agent/pkg/appconfig"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
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
