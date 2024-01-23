/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/dryrun/event"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"time"
)

type (
	logStatSubConsumer struct {
		parent *Consumer
	}
)

func (c *logStatSubConsumer) MaybeFlush() {
}

func (c *logStatSubConsumer) init() {
}

func (c *logStatSubConsumer) setParent(parent *Consumer) {
	c.parent = parent
}

func (c *logStatSubConsumer) Update(f func()) {
	// lock timeline
	timeline := c.parent.timeline
	timeline.Lock()
	defer timeline.Unlock()
	f()
}

func (c *logStatSubConsumer) ProcessGroup(iw *inputWrapper, ctx *LogContext, maxTs *int64) {
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

	groups, b := c.parent.executeGroupBy(ctx)
	if !b {
		if processGroupEvent != nil {
			processGroupEvent.Info("groupBy false, break")
		}
		return
	}

	if processGroupEvent != nil {
		groupMap := make(map[string]string)
		for i, group := range c.parent.task.GroupBy.Groups {
			groupMap[group.Name] = groups[i]
		}
		processGroupEvent.Set("groupBy", groupMap)
	}

	// get data shard
	shard := c.parent.timeline.GetOrCreateShard(alignTs)
	if shard.Frozen {
		c.parent.stat.FilterDelay++
		ctx.periodStatus.Stat.FilterDelay++
		// has log delay there is no need to process it
		return
	}

	point := c.parent.getOrCreateStoragePoint(alignTs, ctx, shard, groups)
	if point == nil {
		return
	}

	c.parent.stat.Processed++
	periodStatus.Stat.Processed++
	c.parent.executeSelectAgg(processGroupEvent, ctx, point)
}

func (c *logStatSubConsumer) Emit(expectedTs int64) bool {
	// TODO 取走数据后给shard打一个标记, 表示已经取走数据了
	// 下次如果还往该shard写数据, 这这些数据是旧的
	// TODO 我们的case里是可以幂等写的!!!

	var datum []*model.DetailData
	c.parent.timeline.Update(func(timeline *storage.Timeline) {
		shard := timeline.GetShard(expectedTs, true)
		if shard == nil {
			logger.Infoz("[consumer] [log] emit nil", //
				zap.String("key", c.parent.key),            //
				zap.Time("ts", time.UnixMilli(expectedTs))) //
			return
		}
		shard.Frozen = true
		points := shard.InternalGetAllPoints()
		for _, v := range points {
			tags := make(map[string]string, len(v.Keys))
			values := make(map[string]interface{}, len(v.Values))
			for i := range v.KeyNames {
				tags[v.KeyNames[i]] = v.Keys[i]
			}
			for i := range v.ValueNames {
				value := v.Values[i]
				switch x := value.(type) {
				// TODO avg case
				// TODO avg是否应该存2个值?
				case *storage.AggNumberDataNode:
					if x.Agg == agg.AggAvg {
						if x.Count > 0 {
							values[v.ValueNames[i]] = x.Value / float64(x.Count)
						} else {
							values[v.ValueNames[i]] = float64(0)
						}
					} else {
						values[v.ValueNames[i]] = x.Value
					}
				default:
					values[v.ValueNames[i]] = 0
				}
			}
			dd := &model.DetailData{
				Timestamp: expectedTs,
				Tags:      tags,
				Values:    values,
			}
			datum = append(datum, dd)

			if len(v.LogSamples) > 0 {
				logSamplesConf := c.parent.task.Select.LogSamples
				if logSamplesConf != nil && logSamplesConf.Enabled {
					dd.Values["logsamples"] = util.ToJsonString(map[string]interface{}{
						"maxCount": c.parent.task.Select.LogSamples.MaxCount,
						"samples": []interface{}{
							map[string]interface{}{
								"hostname": c.parent.getTargetHostname(),
								"logs":     v.LogSamples,
							},
						},
					})
				}
			}

		}

	})

	c.parent.stat.Emit += int32(len(datum))
	c.parent.AddBatchDetailDatus(expectedTs, datum)

	return len(datum) > 0
}
