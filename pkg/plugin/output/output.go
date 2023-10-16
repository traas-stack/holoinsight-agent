/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package output

import "github.com/traas-stack/holoinsight-agent/pkg/model"

type (
	Extension struct {
		Tenant string
	}
	Output interface {
		WriteMetricsV1([]*model.Metric, Extension)

		WriteBatchV4(configKey, targetKey, metricName string, array []*model.DetailData, c *PeriodCompleteness) error
	}
	composite struct {
		array []Output
	}
	PeriodCompleteness struct {
		Valid  bool
		TS     int64
		OK     bool
		Target map[string]string
	}
)

func (c *composite) WriteMetricsV1(metrics []*model.Metric, extension Extension) {
	for _, output := range c.array {
		output.WriteMetricsV1(metrics, extension)
	}
}

func (c *composite) WriteBatchV4(configKey, targetKey, metricName string, array []*model.DetailData, pc *PeriodCompleteness) error {
	for _, output := range c.array {
		output.WriteBatchV4(configKey, targetKey, metricName, array, pc)
	}
	return nil
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
