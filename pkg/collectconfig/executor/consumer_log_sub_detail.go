/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/dryrun/event"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"time"
)

type (
	detailConsumer struct {
		parent *Consumer
		table  *model.Table
	}
)

func (c *detailConsumer) MaybeFlush() {
	if c.table == nil {
		return
	}

	tr := &pb.WriteMetricsRequestV4_TaskResult{
		Key:           c.parent.key,
		RefCollectKey: c.parent.ct.Config.Key,
		RefTargetKey:  c.parent.ct.Target.Key,
		Table: &pb.WriteMetricsRequestV4_Table{
			Timestamp: 0,
			Header: &pb.WriteMetricsRequestV4_Header{
				MetricName: c.parent.metricName,
				TagKeys:    c.table.Header.TagKeys,
				ValueKeys:  c.table.Header.FieldKeys,
			},
		},
		Extension: map[string]string{
			"details": "1",
		},
		Timestamp:    0,
		Completeness: nil,
	}
	for _, row := range c.table.Rows {
		pbValueValues := make([]*pb.DataNode, len(row.FieldValues))
		for i, fieldValue := range row.FieldValues {
			pbValueValues[i] = &pb.DataNode{
				Type:  0,
				Count: 0,
				Value: fieldValue,
				Bytes: nil,
			}
		}
		tr.Table.Rows = append(tr.Table.Rows, &pb.WriteMetricsRequestV4_Row{
			Timestamp:   row.Timestamp,
			TagValues:   row.TagValues,
			ValueValues: nil,
		})
	}

	begin := time.Now()
	err := gateway.GetWriteService().WriteV4(context.Background(), &gateway.WriteV4Request{Batch: []*pb.WriteMetricsRequestV4_TaskResult{tr}})
	sendCost := time.Since(begin)
	logger.Infoz("detail", zap.Int("count", len(c.table.Rows)), zap.Duration("sendCost", sendCost), zap.Error(err))
	c.table = nil
}

func (c *detailConsumer) setParent(parent *Consumer) {
	c.parent = parent
}

func (c *detailConsumer) Update(f func()) {
	f()
}

func (c *detailConsumer) ProcessGroup(iw *inputWrapper, ctx *LogContext, maxTs *int64) {

	var processGroupEvent *event.Event
	if c.parent.debugEvent != nil {
		processGroupEvent = c.parent.debugEvent.AddChild("process group").Set("lines", util.CopyStringSlice(ctx.log.Lines))
		ctx.event = processGroupEvent
	}

	// execute before parse filter
	if processGroupEvent != nil {
		ctx.whereEvent = &event.WhereEvent{}
		ctx.event.Set("beforeParseWhere", ctx.whereEvent)
	}
	if !c.parent.executeBeforeParseWhere(ctx) {
		// 在内部处理了
		return
	}
	ctx.whereEvent = nil

	// execute log parse
	if !c.parent.executeLogParse(ctx) {
		if processGroupEvent != nil {
			processGroupEvent.Info("logParse false, break")
		}
		return
	}

	if !c.parent.executeVarsProcess(ctx) {
		if processGroupEvent != nil {
			processGroupEvent.Info("parseVars error, break")
		}
		return
	}

	// execute time parse
	ts, b := c.parent.executeTimeParse(ctx)
	if !b {
		if processGroupEvent != nil {
			processGroupEvent.Info("parseTime false, break")
		}
		return
	}

	intervalMs := c.parent.Window.Interval.Milliseconds()
	alignTs := ts / intervalMs * intervalMs
	if *maxTs < ts {
		*maxTs = ts
	}

	periodStatus := c.parent.getOrCreatePeriodStatusWithoutLock(alignTs)
	periodStatus.Stat.Broken = periodStatus.Stat.Broken || c.parent.stat.Broken
	periodStatus.Stat.Groups++
	ctx.periodStatus = periodStatus

	if processGroupEvent != nil {
		processGroupEvent.Set("timestamp", ts)
	}

	// execute where
	if processGroupEvent != nil {
		ctx.whereEvent = &event.WhereEvent{}
		ctx.event.Set("where", ctx.whereEvent)
	}
	if !c.parent.executeWhere(ctx) {
		if processGroupEvent != nil {
			processGroupEvent.Info("where false, break")
		}
		return
	}
	ctx.whereEvent = nil

	c.parent.stat.Processed++
	periodStatus.Stat.Processed++

	groupValues, ok := c.parent.executeGroupBy(ctx)
	if !ok {
		return
	}
	xs := c.parent.Select.(*xSelect)

	if c.table == nil {
		c.table = &model.Table{
			Name: c.parent.metricName,
			Header: &model.Header{
				TagKeys:   c.parent.GroupBy.GroupNames(),
				FieldKeys: xs.valueNames,
			},
			Rows: nil,
		}
	}

	row := &model.Row{
		Timestamp:   ts,
		TagValues:   make([]string, len(c.table.Header.TagKeys)),
		FieldValues: make([]float64, len(c.table.Header.FieldKeys)),
	}

	for i := 0; i < len(groupValues); i++ {
		row.TagValues[i] = groupValues[i]
	}

	for i, item := range xs.values {
		if item.agg == agg.AggCount {
			row.FieldValues[i] = 1
			continue
		}
		if item.elect == nil {
			continue
		}
		f64, _ := item.elect.ElectNumber(ctx)
		row.FieldValues[i] = f64
	}

	c.table.Rows = append(c.table.Rows, row)

}

func (c *detailConsumer) Emit(expectedTs int64) bool {
	return true
}

func (c *detailConsumer) init() {
}
