/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package standard

import (
	"github.com/influxdata/telegraf"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"time"
)

type (
	inputWrapper struct {
		input api.Input
	}
)

func (i *inputWrapper) SampleConfig() string {
	return ""
}

func (i *inputWrapper) Gather(accumulator telegraf.Accumulator) error {
	ma := api.NewMemoryAccumulator()
	err := i.input.Collect(ma)
	if err != nil {
		return err
	}

	for i := range ma.Metrics {
		metric := ma.Metrics[i]
		accumulator.AddFields(metric.Name, map[string]interface{}{
			"": metric.Value,
		}, metric.Tags, time.UnixMilli(metric.Timestamp))
	}
	return nil
}

func Wrap(input api.Input) telegraf.Input {
	return &inputWrapper{
		input: input,
	}
}
