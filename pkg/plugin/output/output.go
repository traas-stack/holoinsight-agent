package output

import "github.com/traas-stack/holoinsight-agent/pkg/model"

type (
	Extension struct {
		Tenant string
	}
	Output interface {
		WriteMetrics([]*model.Metric, Extension)

		// 异步地写数据, 不会阻塞
		// 如果异步队列已经满了则返回error
		WriteBatchAsync(configKey, targetKey, metricName string, array []*model.DetailData) error

		// 同步地写数据, 会阻塞到服务端返回
		WriteBatchSync(configKey, targetKey, metricName string, array []*model.DetailData) error

		model.Module
	}
	composite struct {
		array []Output
	}
)

func (c *composite) WriteMetrics(metrics []*model.Metric, extension Extension) {
	for _, output := range c.array {
		output.WriteMetrics(metrics, extension)
	}
}

func (c *composite) WriteBatchAsync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	for _, output := range c.array {
		output.WriteBatchAsync(configKey, targetKey, metricName, array)
	}
	return nil
}

func (c *composite) WriteBatchSync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	for _, output := range c.array {
		output.WriteBatchSync(configKey, targetKey, metricName, array)
	}
	return nil
}

func (c *composite) Start() {
}

func (c *composite) Stop() {
}

func Composite(array ...Output) Output {
	cpy := make([]Output, 0, len(array))
	for _, o := range array {
		if o != nil {
			cpy = append(cpy, o)
		}
	}
	return &composite{array: cpy}
}
