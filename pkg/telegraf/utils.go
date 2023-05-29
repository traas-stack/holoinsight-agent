package telegraf

import (
	telegraf2 "github.com/influxdata/telegraf"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"reflect"
)

func MaybeInitLog(input telegraf2.Input) {
	if input == nil {
		return
	}
	if logField := reflect.ValueOf(input).Elem().FieldByName("Log"); logField.IsNil() {
		logField.Set(reflect.ValueOf(logger.ZapLogger.InfoS))
	}
}
