package accumulator

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
)

type (
	Memory struct {
		Metrics []*model.Metric
	}
)

func (m *Memory) AddMetric(metric *model.Metric) {
	m.Metrics = append(m.Metrics, metric)
}

func (m *Memory) AddError(err error) {
	// noop
}

func (m *Memory) Merge(other *Memory) {
	m.Metrics = append(m.Metrics, other.Metrics...)
}

func (m *Memory) Dump() {
	for _, metric := range m.Metrics {
		fmt.Printf("%s %v %d %f\n", metric.Name, metric.Tags, metric.Timestamp, metric.Value)
	}
}
