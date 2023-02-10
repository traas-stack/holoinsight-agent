package accumulator

import "github.com/traas-stack/holoinsight-agent/pkg/model"

type (
	// Like Telegraf.Accumulator
	Accumulator interface {
		AddMetric(*model.Metric)
		AddError(error)
	}
)
