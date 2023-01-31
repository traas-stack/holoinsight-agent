package console

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/plugin/output"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"time"
)

var Console output.Output = &ConsoleOutput{}

type (
	ConsoleOutput struct{}
)

func (c *ConsoleOutput) WriteMetrics(metrics []*model.Metric, extension output.Extension) {
	for _, metric := range metrics {
		logger.Infof("[output] [console] %s %+v %d %f", metric.Name, metric.Tags, metric.Timestamp, metric.Value)
	}
}

func NewConsoleOutput() (output.Output, error) {
	return Console, nil
}

func (c *ConsoleOutput) WriteBatchAsync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	for _, dd := range array {
		logger.Debugf("[output] [console] write data metric=[%s] ts=[%s] tags=%s values=%s",
			metricName,
			time.UnixMilli(dd.Timestamp).Format(time.RFC3339),
			util.ToJsonString(dd.Tags),
			util.ToJsonString(dd.Values))
	}
	return nil
}

func (c *ConsoleOutput) WriteBatchSync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	return nil
}

func (c *ConsoleOutput) Start() {
}

func (c *ConsoleOutput) Stop() {
}
