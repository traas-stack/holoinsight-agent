package executor

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/TRaaSStack/holoinsight-agent/pkg/loganalysis"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
)

type (
	logAnalysisSubConsumer struct {
		parent *Consumer
		conf   *ParsedConf
	}
	logAnalysisSubConsumerState struct {
		logAnalyzer   *loganalysis.Analyzer
		knownPatterns map[string]*loganalysis.AnalyzedLog
	}
)

func newLogAnalysisSubConsumer(conf *collectconfig.LogAnalysisConf) (*logAnalysisSubConsumer, error) {
	return &logAnalysisSubConsumer{
		conf: parseLogAnalysisConf(conf),
	}, nil
}

func newLogAnalysisSubConsumerState(conf *ParsedConf) *logAnalysisSubConsumerState {
	return &logAnalysisSubConsumerState{
		logAnalyzer:   loganalysis.NewAnalyzer(conf.MaxLogLength, conf.MaxUnknownPatterns),
		knownPatterns: make(map[string]*loganalysis.AnalyzedLog),
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

	// get data shard
	shard := c.parent.timeline.GetOrCreateShard(alignTs)
	if shard.Frozen {
		c.parent.stat.hasLogDelay = true
		// has log delay there is no need to process it
		return
	}

	var state *logAnalysisSubConsumerState
	if shard.Data == nil {
		state = newLogAnalysisSubConsumerState(c.conf)
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
			if t, ok := state.knownPatterns[pattern.Name]; ok {
				t.Count++
			} else {
				t = &loganalysis.AnalyzedLog{Sample: ctx.GetLine(), Count: 1}
				state.knownPatterns[pattern.Name] = t
			}
			return
		}
	}

	state.logAnalyzer.Analyze(ctx.GetLine())
}

func (c *logAnalysisSubConsumer) Emit(expectedTs int64) {
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
		return
	}

	analyzedLogs := state.logAnalyzer.AnalyzedLogs()
	state.logAnalyzer.Clear()

	var metrics []*model.DetailData
	unknownPatternLogsCount := 0
	for _, log := range analyzedLogs {
		unknownPatternLogsCount += log.Count
	}

	totalCount := unknownPatternLogsCount

	knownPatterns := state.knownPatterns
	state.knownPatterns = make(map[string]*loganalysis.AnalyzedLog)
	for pattern, t := range knownPatterns {
		totalCount += t.Count

		r := &loganalysis.Unknown{AnalyzedLogs: []*loganalysis.AnalyzedLog{t}}
		metrics = append(metrics, &model.DetailData{
			Timestamp:   expectedTs,
			Tags:        map[string]string{"eventName": pattern},
			Values:      map[string]interface{}{"value": util.ToJsonString(r)},
			SingleValue: true,
		})
		metrics = append(metrics, &model.DetailData{
			Timestamp:   expectedTs,
			Tags:        map[string]string{"eventName": pattern},
			Values:      map[string]interface{}{"count": t.Count},
			SingleValue: false,
		})
	}

	if totalCount == 0 {
		logger.Debugz("[consumer] [loganalysis] empty logs", zap.String("key", c.parent.key))
		return
	}

	if unknownPatternLogsCount > 0 {
		metrics = append(metrics, &model.DetailData{
			Timestamp: expectedTs,
			Tags:      map[string]string{"eventName": "__analysis"},
			Values: map[string]interface{}{
				"value": util.ToJsonString(&loganalysis.Unknown{
					AnalyzedLogs: analyzedLogs,
				}),
			},
			SingleValue: true,
		})
		metrics = append(metrics, &model.DetailData{
			Timestamp: expectedTs,
			Tags:      map[string]string{"eventName": "__analysis"},
			Values: map[string]interface{}{
				"count": unknownPatternLogsCount,
			},
			SingleValue: false,
		})
	}

	c.parent.AddBatchDetailDatus(metrics)
}
