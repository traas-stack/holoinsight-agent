/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"encoding/gob"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/loganalysis"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
)

type (
	logAnalysisSubConsumer struct {
		parent *Consumer
		conf   *ParsedConf
	}
	logAnalysisSubConsumerState struct {
		LogAnalyzer   *loganalysis.Analyzer
		KnownPatterns map[string]*loganalysis.AnalyzedLog
	}
)

func init() {
	gob.Register(&logAnalysisSubConsumerState{})
}

func newLogAnalysisSubConsumer(conf *collectconfig.LogAnalysisConf) (*logAnalysisSubConsumer, error) {
	return &logAnalysisSubConsumer{
		conf: parseLogAnalysisConf(conf),
	}, nil
}

func newLogAnalysisSubConsumerState(conf *ParsedConf) *logAnalysisSubConsumerState {
	return &logAnalysisSubConsumerState{
		LogAnalyzer:   loganalysis.NewAnalyzer(conf.MaxLogLength, conf.MaxUnknownPatterns),
		KnownPatterns: make(map[string]*loganalysis.AnalyzedLog),
	}
}

func parseLogAnalysisConf(conf *collectconfig.LogAnalysisConf) *ParsedConf {
	parsedPatterns := make([]*ParsedPatternConf, len(conf.Patterns))
	for i := range conf.Patterns {
		w, err := parseWhere(conf.Patterns[i].Where)
		if err != nil {
			logger.Debugz("[consumer] parse LogAnalysis where error", zap.Error(err))
			continue
		}
		parsedPatterns[i] = &ParsedPatternConf{
			LogAnalysisPatternConf: conf.Patterns[i],
			xwhere:                 w,
		}
	}
	return &ParsedConf{
		LogAnalysisConf: conf,
		ParsedPatterns:  parsedPatterns,
	}
}

func (c *logAnalysisSubConsumer) init() {
}

func (c *logAnalysisSubConsumer) setParent(parent *Consumer) {
	c.parent = parent
}

func (c *logAnalysisSubConsumer) Update(f func()) {
	f()
}

func (c *logAnalysisSubConsumer) ProcessGroup(iw *inputWrapper, ctx *LogContext, maxTs *int64) {
	if !c.parent.executeBeforeParseWhere(ctx) {
		return
	}

	// execute time parse
	ts, b := c.parent.executeTimeParse(ctx)
	if !b {
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

	// get data shard
	shard := c.parent.timeline.GetOrCreateShard(alignTs)
	if shard.Frozen {
		c.parent.stat.FilterDelay++
		ctx.periodStatus.Stat.FilterDelay++
		// has log delay there is no need to process it
		return
	}

	c.parent.stat.Processed++
	ctx.periodStatus.Stat.Processed++

	var state *logAnalysisSubConsumerState
	if shard.Data == nil {
		state = newLogAnalysisSubConsumerState(c.conf)
		shard.Data = state
	} else {
		state = shard.Data.(*logAnalysisSubConsumerState)
	}

	for _, pattern := range c.conf.ParsedPatterns {
		ok, err := pattern.xwhere.Test(ctx)
		if err != nil {
			logger.Debugz("[consumer] [loganalysis] pattern where error", zap.Error(err))
			continue
		}
		if ok {
			if t, ok := state.KnownPatterns[pattern.Name]; ok {
				t.Count++
			} else {
				t = &loganalysis.AnalyzedLog{Sample: ctx.GetLine(), Count: 1}
				state.KnownPatterns[pattern.Name] = t
			}
			return
		}
	}

	state.LogAnalyzer.Analyze(ctx.GetLine())
}

func (c *logAnalysisSubConsumer) Emit(expectedTs int64) bool {
	var state *logAnalysisSubConsumerState
	c.parent.timeline.View(func(timeline *storage.Timeline) {
		shard := c.parent.timeline.GetShard(expectedTs)
		if shard == nil {
			return
		}
		shard.Frozen = true
		if shard.Data == nil {
			return
		}
		state = shard.Data.(*logAnalysisSubConsumerState)
		shard.Data = nil
	})

	if state == nil {
		// emit nil 这可能是正常的 比如这一分钟确实没有日志
		logger.Debugz("[consumer] [loganalysis] emit nil", zap.Int64("ts", expectedTs))
		c.parent.AddBatchDetailDatus(expectedTs, nil)
		return false
	}

	analyzedLogs := state.LogAnalyzer.AnalyzedLogs()
	state.LogAnalyzer.Clear()

	var metrics []*model.DetailData
	unknownPatternLogsCount := 0
	for _, log := range analyzedLogs {
		unknownPatternLogsCount += log.Count
	}

	totalCount := unknownPatternLogsCount

	knownPatterns := state.KnownPatterns
	state.KnownPatterns = make(map[string]*loganalysis.AnalyzedLog)
	for pattern, t := range knownPatterns {
		totalCount += t.Count

		r := &loganalysis.Unknown{AnalyzedLogs: []*loganalysis.AnalyzedLog{t}}
		metrics = append(metrics, &model.DetailData{
			Timestamp: expectedTs,
			Tags:      map[string]string{"eventName": pattern},
			Values: map[string]interface{}{
				"value":    t.Count,
				"analysis": util.ToJsonString(r),
			},
		})
	}

	if totalCount == 0 {
		logger.Debugz("[consumer] [loganalysis] empty logs", zap.String("key", c.parent.key))
	}

	if unknownPatternLogsCount > 0 {
		metrics = append(metrics, &model.DetailData{
			Timestamp: expectedTs,
			Tags:      map[string]string{"eventName": "__analysis"},
			Values: map[string]interface{}{
				"analysis": util.ToJsonString(&loganalysis.Unknown{
					AnalyzedLogs: analyzedLogs,
				}),
				"value": unknownPatternLogsCount,
			},
		})
	}

	c.parent.stat.Emit += int32(len(metrics))
	c.parent.AddBatchDetailDatus(expectedTs, metrics)

	return len(metrics) > 0
}
