/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package load

import (
	"github.com/shirou/gopsutil/v3/load"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
)

type (
	loadInput struct {
	}
)

func (l *loadInput) GetDefaultPrefix() string {
	return ""
}

func (l *loadInput) Collect(a api.Accumulator) error {
	loadAvg, err := load.Avg()
	if err != nil {
		return err
	}
	a.AddMetric(&model.Metric{Name: "load_load1", Value: loadAvg.Load1})
	a.AddMetric(&model.Metric{Name: "load_load5", Value: loadAvg.Load5})
	a.AddMetric(&model.Metric{Name: "load_load15", Value: loadAvg.Load15})
	return nil
}
