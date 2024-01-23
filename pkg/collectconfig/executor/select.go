/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
)

const (
	logSamplesMaxCount  = 100
	logSamplesMaxLength = 64 * 1024
)

type (
	DataNode interface {
		GetString() string
		GetCount() int32
		GetNumber() float64
	}
	// Execute select
	XSelect interface {
		Select(ctx *LogContext) ([]DataNode, error)
	}
	xSelect struct {
		valueNames []string
		values     []*xSelectOne
		logSamples *xLogSamples
	}
	xSelectOne struct {
		// TODO 这个地方应该要有类型 否则难搞...
		as    string
		elect XElect
		agg   agg.AggType
		where XWhere
		// TODO 此处可像lego一样携带一个where
	}
	DataNodeImpl struct {
		String string
		Value  float64
		Count  int32
	}
	xLogSamples struct {
		Where     XWhere
		MaxCount  int
		MaxLength int
	}
)

func (d *DataNodeImpl) GetString() string {
	return d.String
}

func (d *DataNodeImpl) GetCount() int32 {
	return d.Count
}

func (d *DataNodeImpl) GetNumber() float64 {
	return d.Value
}

func (x *xSelect) Select(ctx *LogContext) ([]DataNode, error) {
	nodes := make([]DataNode, len(x.values))
	for i, value := range x.values {
		// TODO value的类型必须要确定
		e, err := value.elect.Elect(ctx)
		if err != nil {
			return nil, err
		}
		n := &DataNodeImpl{}
		switch c := e.(type) {
		case string:
			n.String = c
		case float64:
			n.Value = c
		}
		nodes[i] = n
	}
	return nodes, nil
}

func parseSelect(s *collectconfig.Select) (XSelect, error) {
	if s == nil {
		return nil, errors.New("collectconfig.Select is nil")
	}
	valueNames := make([]string, len(s.Values))
	values := make([]*xSelectOne, len(s.Values))
	for i, so := range s.Values {
		aggType := agg.GetAggType(so.Agg)
		if aggType == agg.AggUnknown {
			return nil, errors.New("AggUnknown")
		}
		var elect XElect
		var err error
		if aggType != agg.AggCount && aggType != agg.AggLogAnalysis {
			elect, err = parseElect(so.Elect)
			if err != nil {
				return nil, err
			}
		}
		where, err := parseWhere(so.Where)
		if err != nil {
			return nil, err
		}
		valueNames[i] = so.As
		values[i] = &xSelectOne{
			as:    so.As,
			elect: elect,
			agg:   aggType,
			where: where,
		}
	}

	var logSamples *xLogSamples
	if s.LogSamples != nil && s.LogSamples.Enabled {
		if ls, err := parseLogSamples(s.LogSamples); err != nil {
			return nil, err
		} else {
			logSamples = ls
		}
	}

	return &xSelect{
		valueNames: valueNames,
		values:     values,
		logSamples: logSamples,
	}, nil
}

func parseLogSamples(c *collectconfig.LogSamples) (*xLogSamples, error) {
	if c == nil {
		return nil, nil
	}
	where, err := parseWhere(c.Where)
	if err != nil {
		return nil, err
	}
	if c.MaxCount > logSamplesMaxCount {
		c.MaxCount = logSamplesMaxCount
	}
	if c.MaxLength > logSamplesMaxLength {
		c.MaxLength = logSamplesMaxLength
	}
	return &xLogSamples{
		Where:     where,
		MaxCount:  c.MaxCount,
		MaxLength: c.MaxLength,
	}, nil
}
