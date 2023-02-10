package sys

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output/console"
	"go.uber.org/zap"
	"strings"
)

type (
	Consumer struct {
		out     output.Output
		ct      *collecttask.CollectTask
		sqlTask *collectconfig.SQLTask
	}
)

func ParseConsumer(ct *collecttask.CollectTask, sqlTask *collectconfig.SQLTask) (*Consumer, error) {
	// TODO config
	out, err := output.Parse(sqlTask.Output.Type, sqlTask.Output)
	if err != nil {
		return nil, err
	}

	out = output.Composite(out, console.Console)

	return &Consumer{
		out:     out,
		ct:      ct,
		sqlTask: sqlTask,
	}, nil
}

func (c *Consumer) Consume(array []*model.DetailData) {
	metricName := c.ct.Config.Key

	// 这个优先级更高
	if c.sqlTask.Output.Gateway != nil && c.sqlTask.Output.Gateway.MetricName != "" {
		// 可以按用户给定的format?
		metricName = c.sqlTask.Output.Gateway.MetricName
	}
	if strings.HasSuffix(metricName, c.sqlTask.From.Type) {
		metricName = metricName[:len(metricName)-len(c.sqlTask.From.Type)]
	}
	logger.Infoz("WriteBatchAsync", zap.String("key", c.ct.Key), zap.String("metricName", metricName))

	// TODO metrics name是在哪里决定的 比较合适?
	// TODO 这里我先异步写了, 如果有重要数据要同步写
	// TODO 在哪里重试?
	err := c.out.WriteBatchAsync(c.ct.Config.Key, c.ct.Target.Key, metricName, array)
	if err != nil {
		// 一般是 write buffer full 比较严重的
		logger.Errorz("WriteBatchAsync error",
			zap.String("config", c.ct.Config.Key),
			zap.String("target", c.ct.Target.Key),
			zap.Error(err))
	}
}

func (c *Consumer) Start() {
	if c.out != nil {
		c.out.Start()
	}
}

func (c *Consumer) Stop() {
	if c.out != nil {
		c.out.Stop()
	}
}
