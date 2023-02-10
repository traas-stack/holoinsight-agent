package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"runtime"
	"strings"
)

func parseConsumer(st *api.SubTask) (*Consumer, error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			logger.Errorf("[consumer] parse consume error %s: %v\n%s", st.CT.Key, r, string(buf))
		}
	}()

	task := st.SqlTask
	xselect, err := parseSelect(task.Select)
	if err != nil {
		return nil, err
	}

	xwhere, err := parseWhere(task.Where)
	if err != nil {
		return nil, err
	}

	xgroupBy, err := parseGroupBy(task.GroupBy)
	if err != nil {
		return nil, err
	}

	if task.Window == nil {
		return nil, errors.New("window is nil")
	}
	interval, err := util.ParseDuration(task.Window.Interval)
	if err != nil {
		return nil, err
	}
	xWindow := &XWindow{
		Interval: interval,
	}

	var logParser LogParser
	var timeParser TimeParser
	var beforeParseWhere XWhere
	var multiline *xMultiline
	if parse := task.From.Log.Parse; parse != nil {
		if logParser, err = parseLogParser(parse); err != nil {
			return nil, err
		}
		if beforeParseWhere, err = parseWhere(parse.Where); err != nil {
			return nil, err
		}
	}
	if timeParser, err = parseTimeParser(task.From.Log.Time); err != nil {
		return nil, err
	}
	if multiline, err = parseMultiline(task.From.Log.Multiline); err != nil {
		return nil, err
	}

	metricName := st.CT.Config.Key
	if task.Output.Gateway != nil && task.Output.Gateway.MetricName != "" {
		metricName = task.Output.Gateway.MetricName
	}
	if strings.HasSuffix(metricName, task.From.Type) {
		metricName = metricName[:len(metricName)-len(task.From.Type)]
	}

	logger.Infoz("[consumer] [init]", //
		zap.String("key", st.CT.Key), //
		zap.String("metricName", metricName))

	var multilineAccumulator *multilineAccumulator
	if multiline != nil {
		multilineAccumulator = newMultilineAccumulator(multiline)
	}

	var sub SubConsumer

	if task.GroupBy.LogAnalysis != nil {
		sub, err = newLogAnalysisSubConsumer(task.GroupBy.LogAnalysis)
		if err != nil {
			return nil, err
		}
	} else {
		sub = &logStatSubConsumer{}
	}

	c := &Consumer{
		key:                  st.CT.Key,
		ct:                   st.CT,
		metricName:           metricName,
		task:                 task,
		Select:               xselect,
		Where:                xwhere,
		GroupBy:              xgroupBy,
		Window:               xWindow,
		LogParser:            logParser,
		TimeParser:           timeParser,
		BeforeParseWhere:     beforeParseWhere,
		multilineAccumulator: multilineAccumulator,
		stopSignal:           util.NewStopSignal(),
		sub:                  sub,
	}

	if sub != nil {
		sub.setParent(c)
	}

	c.init()
	return c, nil
}
