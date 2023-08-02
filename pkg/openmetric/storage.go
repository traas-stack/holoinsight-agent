/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package openmetric

import (
	"context"
	prometheusmodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/exemplar"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output/gateway"
	"go.uber.org/zap"
	"math"
	"time"

	"github.com/prometheus/prometheus/storage"
)

type (
	Storage struct {
	}
	entry struct {
		l labels.Labels
		t int64
		v float64
	}
	storageAppender struct {
		buffer []*entry
	}
)

func (s *Storage) Appender(ctx context.Context) storage.Appender {
	return &storageAppender{}
}

func (s *storageAppender) Append(ref uint64, l labels.Labels, t int64, v float64) (uint64, error) {
	s.buffer = append(s.buffer, &entry{
		l: l,
		t: t,
		v: v,
	})
	return 0, nil
}

func (s *storageAppender) AppendExemplar(ref uint64, l labels.Labels, e exemplar.Exemplar) (uint64, error) {
	return 0, nil
}

func (s *storageAppender) Commit() error {
	if len(s.buffer) == 0 {
		return nil
	}

	metrics := make([]*model.Metric, 0, len(s.buffer))
	for _, e := range s.buffer {
		if math.IsNaN(e.v) {
			continue
		}
		name := ""
		tags := make(map[string]string, len(e.l))
		for _, label := range e.l {
			if label.Name == prometheusmodel.MetricNameLabel {
				name = label.Value
			} else {
				tags[label.Name] = label.Value
			}
		}
		metrics = append(metrics, &model.Metric{
			Name:      name,
			Tags:      tags,
			Timestamp: e.t,
			Value:     e.v,
		})

		if logger.DebugEnabled {
			logger.Debugz("[openmetrics] [storage] write metric", zap.Int64("t", e.t), zap.Any("labels", e.l), zap.Float64("v", e.v))
		}
	}

	// TODO timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := gateway.GetWriteService().WriteV1(ctx, &gateway.WriteV1Request{
		Batch: metrics,
	})
	logger.Infoz("[openmetrics] [storage] write", zap.Int("size", len(metrics)), zap.Error(err))
	s.buffer = nil

	return err
}

func (s *storageAppender) Rollback() error {
	s.buffer = nil
	return nil
}
