/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package telegraf

import (
	"fmt"
	"github.com/influxdata/telegraf"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
)

type (
	// InputAdapter adapts telegraf.Input to api.Input
	InputAdapter struct {
		defaultPrefix string
		input         telegraf.Input
	}
)

func NewInputAdapter(input telegraf.Input) api.Input {
	return &InputAdapter{defaultPrefix: "telegraf_", input: input}

}
func (i *InputAdapter) GetDefaultPrefix() string {
	return i.defaultPrefix
}

func (i *InputAdapter) Collect(a api.Accumulator) error {
	memory := &Memory{}
	if err := i.input.Gather(memory); err != nil {
		return err
	}

	for _, metric := range memory.Metrics {
		for key, value := range metric.Fields() {
			if f64, err := cast.ToFloat64E(value); err == nil {
				a.AddMetric(&model.Metric{
					Name: fmt.Sprintf("%s_%s", metric.Name(), key),
					Tags: metric.Tags(),
					// Timestamp is filled by pipeline layer.
					Value: f64,
				})
			}
		}
	}

	return nil
}
