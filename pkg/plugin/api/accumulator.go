/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package api

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"io"
)

type (
	Accumulator interface {
		AddMetric(*model.Metric)
	}

	MemoryAccumulator struct {
		Metrics []*model.Metric
	}
	prefixAccumulator struct {
		a      Accumulator
		prefix string
	}
)

func NewMemoryAccumulator() *MemoryAccumulator {
	return &MemoryAccumulator{}
}

func (ma *MemoryAccumulator) AddMetric(metric *model.Metric) {
	ma.Metrics = append(ma.Metrics, metric)
}

func (ma *MemoryAccumulator) PrintTo(writer io.Writer) error {
	for i := range ma.Metrics {
		_, err := fmt.Fprintf(writer, "%v\n", ma.Metrics[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func WithPrefix(a Accumulator, prefix string) Accumulator {
	return &prefixAccumulator{
		a:      a,
		prefix: prefix,
	}
}

func (p *prefixAccumulator) AddMetric(metric *model.Metric) {
	metric.Name = p.prefix + metric.Name
	p.a.AddMetric(metric)
}
