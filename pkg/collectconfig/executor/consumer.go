/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	json2 "encoding/json"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/dryrun/event"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
	pb2 "github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"math"
	"os"
	"strings"
	"time"
)

const (
	hardMaxKeySize = 100_000
	// ReportTaskInfoInterval indicates the time interval of reporting task info.
	ReportTaskInfoInterval = 10
)

type (
	// Consumer 的设计是不用锁的, 它完全靠上层(Pipeline)来调度, 由 Pipeline 负责保证 Consumer 的调用是安全的
	Consumer struct {
		ct          *collecttask.CollectTask
		key         string
		storage     *storage.Storage
		timeline    *storage.Timeline
		task        *collectconfig.SQLTask
		output      output.Output
		pointKeyBuf []byte

		Select               XSelect
		Where                XWhere
		GroupBy              XGroupBy
		Window               *XWindow
		LogParser            LogParser
		TimeParser           TimeParser
		varsProcessor        *varsProcessor
		BeforeParseWhere     XWhere
		stat                 consumerStat
		multilineAccumulator *multilineAccumulator

		//lastAlignTs     int64
		stopSignal *util.StopSignal

		// 用于Stop方法, 如果为true表示该Consumer是因为配置更新而被 stop, 会有另外一个 Consumer start 代替它的功能
		updated bool

		tsWalker   *util.AlignTsWalker
		metricName string

		// 实际遇到过的最大时间戳
		maxDataTimestamp int64
		// estimatedMaxDataTimestamp is an estimate of the actual log time.
		// When there is no logs in file this period, this estimatedMaxDataTimestamp value will be treated as maxDataTimestamp as if there is really a log at this time.
		estimatedMaxDataTimestamp int64

		maxKeySize int
		sub        SubConsumer

		debugEvent *event.Event
		runInLock  func(f func())

		// printStatCalledCounter records the number of calls to `printStat`
		printStatCalledCounter int
		firstIOSuccessTime     int64
	}

	consumerStat struct {
		ioTotal int32
		ioError int32
		// success情况下的空拉次数
		ioEmpty int32
		// 涉及的流量
		bytes int64
		// 日志行数
		lines int32
		// 日志组数, 多行情况下 一组多行日志算为一个组
		groups int32
		broken bool
		// 文件不存在
		miss                   bool
		noContinued            bool
		processed              int32
		filterBeforeParseWhere int32
		filterLogParseError    int32
		filterTimeParseError   int32
		filterWhere            int32
		filterGroup            int32
		filterGroupMaxKeys     int32
		filterIgnore           int32
		filterMultiline        int32
		filterDelay            int32
		emit                   int32
		emitSuccess            int32
		emitError              int32
		// error count when agg
		aggWhereError int32
		// error count when select
		selectError int32
	}

	ParsedConf struct {
		*collectconfig.LogAnalysisConf
		ParsedPatterns []*ParsedPatternConf
	}
	ParsedPatternConf struct {
		*collectconfig.LogAnalysisPatternConf
		// parsed where
		xwhere XWhere
	}
	EmitContext struct{}

	DataAccumulator interface {
		AddBatchDetailDatus([]*model.DetailData)
	}

	// PeriodStatus holds stats for that time period
	PeriodStatus struct {
		stat                      consumerStat
		EmitSuccess               bool
		EmitError                 int
		estimatedMaxDataTimestamp int64
	}
)

func (c *Consumer) reportUpEvent(expectedTs int64, ps *PeriodStatus) {
	// ok means data in expectedTs time window is complete, the following conditions must be met:
	// 1. emit success
	// 2. log consumption is not lagging behind
	// 3. the task is not started in the middle of the cycle
	ok := ps.EmitSuccess &&
		ps.estimatedMaxDataTimestamp >= expectedTs+c.Window.Interval.Milliseconds() &&
		c.firstIOSuccessTime < expectedTs

	event := &pb2.ReportEventRequest_Event{
		BornTimestamp:  time.Now().UnixMilli(),
		EventTimestamp: expectedTs,
		EventType:      "STAT",
		PayloadType:    "log_monitor_up",
		Tags:           c.getCommonEventTags(),
		Numbers: map[string]int64{
			// up==1 means task is running
			"up": 1,
			"ok": util.BoolToInt64(ok), //

			"in_groups":    int64(ps.stat.groups),
			"in_processed": int64(ps.stat.processed),

			"f_where": int64(ps.stat.filterWhere),
			"f_group": int64(ps.stat.filterGroup),
			"f_gkeys": int64(ps.stat.filterGroupMaxKeys),
			"f_delay": int64(ps.stat.filterDelay),

			"p_select": int64(ps.stat.selectError),
		},
		Strings: nil,
	}
	removeZeroNumbers(event)
	ioc.RegistryService.ReportEventAsync(event)
}

func (c *Consumer) updatePeriodStatus(expectedTs int64, callback func(status *PeriodStatus)) {
	c.timeline.Update(func(timeline *storage.Timeline) {
		c.updatePeriodStatusWithoutLock(expectedTs, callback)
	})
}

func (c *Consumer) updatePeriodStatusWithoutLock(expectedTs int64, callback func(status *PeriodStatus)) {
	shard := c.timeline.GetOrCreateShard(expectedTs)
	if shard.Data2 == nil {
		shard.Data2 = &PeriodStatus{}
	}
	callback(shard.Data2.(*PeriodStatus))
}

func (c *Consumer) getOrCreatePeriodStatusWithoutLock(expectedTs int64) *PeriodStatus {
	shard := c.timeline.GetOrCreateShard(expectedTs)
	if shard.Data2 == nil {
		shard.Data2 = &PeriodStatus{}
	}
	return shard.Data2.(*PeriodStatus)
}

func (c *Consumer) reportLogs(eventTs int64, logs ...string) {
	event := &pb2.ReportEventRequest_Event{
		EventTimestamp: eventTs,
		EventType:      "DIGEST",
		PayloadType:    "log_conumser_digest",
		Tags:           c.getCommonEventTags(),
		Logs:           logs,
	}
	ioc.RegistryService.ReportEventAsync(event)
}

func (c *Consumer) AddBatchDetailDatus(expectedTs int64, datum []*model.DetailData) {
	if len(datum) == 0 {
		c.updatePeriodStatus(expectedTs, func(status *PeriodStatus) {
			status.EmitSuccess = true
			c.reportUpEvent(expectedTs, status)
		})
		return
	}

	if !c.addCommonTags(datum) {
		logger.Errorz("[log] [consumer] fail to add common tags to metrics", zap.String("key", c.key))
		return
	}

	go func() {
		err := c.output.WriteBatchSync(c.ct.Config.Key, c.ct.Target.Key, c.metricName, datum)
		c.runInLock(func() {
			c.updatePeriodStatus(expectedTs, func(status *PeriodStatus) {
				if err != nil {
					status.EmitError += len(datum)
				} else {
					status.EmitSuccess = true
				}
				c.reportUpEvent(expectedTs, status)
			})

			if err == nil {
				c.stat.emitSuccess += int32(len(datum))
			} else {
				c.stat.emitError += int32(len(datum))
				c.reportLogs(expectedTs, fmt.Sprintf("emit error %+v", err))
				logger.Errorz("[log] [consumer] emit error", zap.String("key", c.key), zap.Error(err))
			}
		})
	}()
}

func (c *Consumer) Consume(resp *logstream.ReadResponse, iw *inputWrapper, err error) {
	c.stat.ioTotal++
	if err != nil {
		c.stat.ioError++
		logger.Errorz("[consumer] [log] digest",
			zap.String("key", c.key),
			zap.Any("resp", resp),
			zap.Error(err))
		return
	}
	fileNotExists := os.IsNotExist(resp.Error)
	if fileNotExists {
		// There is no need to stat ioError when file misses.
		// This is helpful to reduce server log size.
		c.stat.miss = true
		logger.Infoz("[consumer] [log] digest, file not exist", //
			zap.String("key", c.key),    //
			zap.String("path", iw.path), //
		)
		return
	}

	if c.firstIOSuccessTime == 0 {
		c.firstIOSuccessTime = resp.IOStartTime.UnixMilli()
	}

	// 说明end不完整
	if resp.HasBroken {
		c.stat.broken = true
	}
	if !resp.Continued {
		c.stat.noContinued = true
	}
	c.stat.bytes += resp.Bytes()
	c.stat.lines += int32(len(resp.Lines))
	if len(resp.Lines) == 0 {
		c.stat.ioEmpty++
	}

	maxTs := c.consume(resp, iw)
	if maxTs < 0 {
		maxTs = 0
	}
	if c.maxDataTimestamp < maxTs {
		c.maxDataTimestamp = maxTs
	}

	if c.estimatedMaxDataTimestamp < c.maxDataTimestamp {
		c.estimatedMaxDataTimestamp = c.maxDataTimestamp
	}

	// 'HasMore == true' means there is more logs, we will to start next pulling as soon as possible.
	// Next time when 'HasMore == false' we will update the estimatedMaxDataTimestamp.
	if !resp.HasMore {
		// 'HasMore == false' means we have reached the log file end.
		// So we can safely update the estimatedMaxDataTimestamp to 'IOStartTime - logDelayTolerance'.
		ts := resp.IOStartTime.Add(-logDelayTolerance).UnixMilli()
		if c.estimatedMaxDataTimestamp < ts {
			c.estimatedMaxDataTimestamp = ts
		}
	}

	logger.Infoz("[consumer] [log] digest", //
		zap.String("key", c.key),                    //
		zap.String("path", resp.Path),               //
		zap.Int64("beginOffset", resp.BeginOffset),  //
		zap.Int64("endOffset", resp.EndOffset),      //
		zap.Bool("continued", resp.Continued),       //
		zap.Bool("more", resp.HasMore),              //
		zap.String("fileId", resp.FileId),           //
		zap.Time("dataTime", time.UnixMilli(maxTs)), //
		zap.Error(resp.Error),                       //
	)
}

func (c *Consumer) SetStorage(s *storage.Storage) {
	c.storage = s

	var timeline *storage.Timeline
	timelineKey := c.key
	s.View(func(_ *storage.Storage) {
		timeline = s.GetTimeline(timelineKey)
		if timeline != nil {
			timeline.Update(func(t *storage.Timeline) {
				t.AddRef(1)
			})
		}
	})

	intervalMs := c.Window.Interval.Milliseconds()
	if timeline == nil {
		capacity := getCapacity(intervalMs)
		logger.Infoz("[consumer] create timeline",
			zap.String("key", c.key),
			zap.Int64("interval", intervalMs),
			zap.Int64("capacity", capacity))
		timeline = storage.NewTimeline(timelineKey, intervalMs, capacity)
		s.Update(func(storage *storage.Storage) {
			s.SetTimeline(timelineKey, timeline)
		})
	}

	c.timeline = timeline
}

func getCapacity(interval int64) int64 {
	switch interval {
	case 1000:
		return 60
	case 5000:
		return 36
	default:
		return 5
	}
}

func (c *Consumer) Start() {
	ioc.RegistryService.ReportEventAsync(&pb2.ReportEventRequest_Event{
		BornTimestamp:  time.Now().UnixMilli(),
		EventTimestamp: time.Now().UnixMilli(),
		EventType:      "DIGEST",
		PayloadType:    "log_monitor_start",
		Tags:           c.getCommonEventTags(),
	})
	logger.Infoz("[consumer] start", zap.String("key", c.key), zap.String("version", c.ct.Version))
	if c.task.Output != nil {
		out, err := output.Parse(c.task.Output.Type, c.task.Output)
		if err != nil {
			logger.Errorz("[consumer] parse output error",
				zap.String("key", c.key),
				zap.String("outputType", c.task.Output.Type),
				zap.Error(err))
			return
		}
		out.Start()
		c.SetOutput(out)
	}
}

func (c *Consumer) maybeReleaseTimeline() {
	if c.storage != nil && c.timeline != nil {
		c.timeline.Update(func(t *storage.Timeline) {
			if t.AddRef(-1) == 0 {
				c.storage.Update(func(s *storage.Storage) {
					s.DeleteTimeline(t.Key)
				})
			}
		})
	}
}

func (c *Consumer) Stop() {
	ioc.RegistryService.ReportEventAsync(&pb2.ReportEventRequest_Event{
		BornTimestamp:  time.Now().UnixMilli(),
		EventTimestamp: time.Now().UnixMilli(),
		EventType:      "DIGEST",
		PayloadType:    "log_monitor_stop",
		Tags:           c.getCommonEventTags(),
		Strings: map[string]string{
			"c_content": string(c.ct.Config.Content),
			"c_version": c.ct.Config.Version,
		},
	})
	logger.Infoz("[consumer] stop", zap.String("key", c.key), zap.String("version", c.ct.Version))

	if !c.updated {
		c.maybeReleaseTimeline()
	}

	if c.output != nil {
		c.output.Stop()
	}
}

func (c *Consumer) Update(o *Consumer) {
	logger.Infoz("[consumer] [log] update", zap.String("consumer", c.key))
	ioc.RegistryService.ReportEventAsync(&pb2.ReportEventRequest_Event{
		BornTimestamp:  time.Now().UnixMilli(),
		EventTimestamp: time.Now().UnixMilli(),
		EventType:      "DIGEST",
		PayloadType:    "log_monitor_update",
		Tags:           c.getCommonEventTags(),
		Strings: map[string]string{
			"c_old_version": o.ct.Config.Version,
			"c_old_content": string(o.ct.Config.Content),
			"c_new_version": c.ct.Config.Version,
			"c_new_content": string(c.ct.Config.Content),
		},
	})

	// 继承一些属性 这样状态不会丢失
	c.stat = o.stat

	// 检查时间窗口是否变化
	if o.Window.Interval != c.Window.Interval {
		logger.Infoz("[consumer] window changed, delete old timeline", zap.String("key", c.key))
		o.storage.Update(func(s *storage.Storage) {
			s.DeleteTimeline(o.timeline.Key)
		})
	} else {
		c.tsWalker = o.tsWalker
		c.storage = o.storage
		c.timeline = o.timeline
		// TODO when file path changed, the firstIOSuccessTime need to be reset to zero
		c.firstIOSuccessTime = o.firstIOSuccessTime
		c.printStatCalledCounter = o.printStatCalledCounter
	}

	// 设置为true, 表明它已经被继承了, 此后stop时不要删除timeline之类的
	o.updated = true
}

// 返回的string不能被长期持有, 它必须最终释放
// TODO 这个方法的实现有一个缺点, 它会长期持有一个 pointKey
// 当程序运行够久之后, pointKey底层占据的内存取向稳定, 比如1KB
// 如果 pipeline 有明显的批处理情况, 那么就可以将pointKey的生命周期缩到批量
// 此时既可以提升性能, 又不会对内存造成太大影响
func (c *Consumer) joinPointKey(groups []string) string {
	b := c.pointKeyBuf
	for _, g := range groups {
		b = append(b, g...)
		b = append(b, ',')
	}
	pointKey := util.String(b)
	c.pointKeyBuf = b[:0]
	return pointKey
}

// Get target time for parse timestramp
func (c *Consumer) getTargetTimezone() *time.Location {
	crii := ioc.Crii
	if crii == nil {
		return nil
	}
	if c.ct.Target.IsTypePod() {
		if pod, ok := crii.GetPod(c.ct.Target.GetNamespace(), c.ct.Target.GetPodName()); ok {
			if mb := pod.MainBiz(); mb != nil && mb.EtcLocaltime != "" {
				if tz, err := time.LoadLocation(mb.EtcLocaltime); err == nil {
					return tz
				}
			}
		}
	}
	return nil
}

func (c *Consumer) getTargetPod() *cri.Pod {
	crii := ioc.Crii
	if crii == nil {
		return nil
	}
	if c.ct.Target.IsTypePod() {
		if pod, ok := crii.GetPod(c.ct.Target.GetNamespace(), c.ct.Target.GetPodName()); ok {
			return pod
		}
	}
	return nil
}

func (c *Consumer) consume(resp *logstream.ReadResponse, iw *inputWrapper) int64 {
	maxTs := int64(-1)
	c.sub.Update(func() {
		c.processMultiline(iw, resp, func(ctx *LogContext) {
			c.sub.ProcessGroup(iw, ctx, &maxTs)
		})
	})
	return maxTs
}

func (c *Consumer) processMultiline(iw *inputWrapper, resp *logstream.ReadResponse, consumer func(*LogContext)) {
	ctx := &LogContext{
		tz: c.getTargetTimezone(),
	}
	oneLine := &LogGroup{Lines: []string{""}}
	var err error

	for i := 0; i <= len(resp.Lines); i++ {
		var fullGroup *LogGroup

		// 特殊处理
		if i == len(resp.Lines) {
			if c.multilineAccumulator == nil {
				// 没有多行模式, 不需要
				continue
			}

			// 进入这个block说明已经遍历完当前批次了, 此时可能有一种特殊情况: pendingLog 有残留 lines
			// 但此时无法知道该组是否已经完整了, 因为组的完成性是靠遇到下一个组的 start 才能确定的, 如果下一行首日志迟迟不打印呢? 或者由于各种原因, 暂时还没有读到下一行首日志呢

			// 如果还有更多日志, 那么我们跳过本次操作让下次的读取来驱动 pendingLog 的 finish 即可
			pending := c.multilineAccumulator.getAndClearPending()
			// pending == nil means no more pending logs
			if pending == nil || resp.HasMore {
				continue
			}

			// Here, means we have pending logs

			// 否则可以断定当前就是读到日志尾了, 且tempLog里有日志残留, 它们是否自成一组呢?
			// 我们现在认为, 一旦读到文件尾了(hasMore==false), 那么最后一组一定自成一组, 虽然这个假设在极端情况下不成立

			fullGroup = pending
		} else {
			line := resp.Lines[i]
			oneLine.SetOneLine(line)
			if c.multilineAccumulator != nil {
				ctx.log = oneLine

				fullGroup, err = c.multilineAccumulator.add(ctx)
				if err != nil {
					logger.Debugz("[consumer] parse multiline error", zap.String("consumer", c.key), zap.Error(err))
					c.stat.filterMultiline++
					continue
				}
				if fullGroup == nil {
					continue
				}
			} else {
				// 不需要配置多行 默认过滤掉一些常见的java error case
				if line == "" || strings.HasPrefix(line, "\tat") || (strings.HasPrefix(line, "\t... ") && strings.HasSuffix(line, " more")) {
					c.stat.filterIgnore++
					continue
				}
				fullGroup = oneLine
			}
		}
		ctx.log = fullGroup
		ctx.path = iw.path
		ctx.pathTags = iw.pathTags
		c.stat.groups++
		consumer(ctx)
		ctx.clearData()
	}
}

func (c *Consumer) createPoint(alignTs int64, groupKeyValues []string) *storage.Point {
	xs := c.Select.(*xSelect)
	values := make([]interface{}, len(xs.values))
	for i, so := range xs.values {
		switch so.agg {
		case agg.AggHll:
			values[i] = storage.NewHllDataNode()
		default:
			values[i] = storage.NewAggNumberDataNode(so.agg)
		}
	}
	return &storage.Point{
		Timestamp:  alignTs,
		KeyNames:   c.GroupBy.GroupNames(),
		Keys:       groupKeyValues,
		ValueNames: xs.valueNames,
		Values:     values,
	}
}

func (c *Consumer) getCommonEventTags() map[string]string {
	tags := map[string]string{
		"agent":       agentmeta.GetAgentId(),
		"t_key":       c.ct.Key,
		"t_c_key":     c.ct.Config.Key,
		"t_c_version": c.ct.Config.Version,
	}

	if pod := c.getTargetPod(); pod != nil {
		tags["t_ip"] = pod.IP()
		tags["t_agentIP"] = util.GetLocalIp()
	} else {
		tags["t_ip"] = util.GetLocalIp()
	}
	return tags
}

func (c *Consumer) createTaskInfoEvent(stat consumerStat) *pb2.ReportEventRequest_Event {
	parsedContent := make(map[string]interface{})
	json2.Unmarshal(c.ct.Config.Content, &parsedContent)

	json := map[string]interface{}{
		"t_key":    c.key,
		"in_mdt":   c.maxDataTimestamp,
		"in_emdt":  c.estimatedMaxDataTimestamp,
		"c_config": parsedContent,
		"c_target": c.ct.Target,
	}

	now := time.Now().UnixMilli()
	if !stat.miss {
		lag := (now - c.estimatedMaxDataTimestamp) / 1000
		if lag > 30 {
			json["in_lag"] = (now - c.estimatedMaxDataTimestamp) / 1000
		}
	}

	if pod := c.getTargetPod(); pod != nil {
		if c := pod.MainBiz(); c != nil {
			json["t_tz"] = c.EtcLocaltime
		}
	} else {
		json["t_tz"] = util.GetLocalTimezone()
	}

	return &pb2.ReportEventRequest_Event{
		BornTimestamp: now,
		EventType:     "DIGEST",
		PayloadType:   "log_monitor_task_info",
		Tags:          c.getCommonEventTags(),
		Json:          util.ToJsonString(json),
	}
}

func (c *Consumer) createStatEvent(stat consumerStat) *pb2.ReportEventRequest_Event {
	event := &pb2.ReportEventRequest_Event{
		BornTimestamp: time.Now().UnixMilli(),
		EventType:     "STAT",
		PayloadType:   "log_monitor_stat",
		Tags:          c.getCommonEventTags(),
		Numbers: map[string]int64{
			"in_io_error": int64(stat.ioError),

			"in_bytes":  stat.bytes,
			"in_lines":  int64(stat.lines),
			"in_groups": int64(stat.groups),

			"in_miss": util.BoolToInt64(stat.miss),

			"in_broken":    util.BoolToInt64(stat.broken),
			"in_skip":      util.BoolToInt64(stat.noContinued),
			"in_processed": int64(stat.processed),

			"f_logparse":  int64(stat.filterLogParseError),
			"f_ignore":    int64(stat.filterIgnore),
			"f_timeparse": int64(stat.filterTimeParseError),
			"f_bwhere":    int64(stat.filterBeforeParseWhere),
			"f_group":     int64(stat.filterGroup),
			"f_gkeys":     int64(stat.filterGroupMaxKeys),
			"f_where":     int64(stat.filterWhere),
			"f_delay":     int64(stat.filterDelay),
			"f_multiline": int64(stat.filterMultiline),

			"out_emit":  int64(stat.emit),
			"out_error": int64(stat.emitError),

			"p_agg":    int64(stat.aggWhereError),
			"p_select": int64(stat.selectError),
		},
		Strings: map[string]string{},
	}
	removeZeroNumbers(event)
	return event
}

func (c *Consumer) printStat() {
	stat := c.stat
	c.stat = consumerStat{}

	{
		events := []*pb2.ReportEventRequest_Event{c.createStatEvent(stat)}
		if c.printStatCalledCounter == 0 {
			events = append(events, c.createTaskInfoEvent(stat))
		}
		ioc.RegistryService.ReportEventAsync(events...)
		c.printStatCalledCounter++
		if c.printStatCalledCounter > ReportTaskInfoInterval {
			c.printStatCalledCounter = 0
		}
	}

	logger.Infoz("[consumer] [log] stat", //
		zap.String("key", c.key), //
		zap.String("configKey", c.ct.Config.Key+"/"+c.ct.Config.Version), //
		zap.String("targetKey", c.ct.Target.Key+"/"+c.ct.Target.Version), //
		zap.Int32("ioEmpty", stat.ioEmpty),                               //
		zap.Int32("ioError", stat.ioError),                               //
		zap.Int32("ioTotal", stat.ioTotal),                               //
		zap.Int64("bytes", stat.bytes),
		zap.Int32("lines", stat.lines),
		zap.Int32("groups", stat.groups),
		zap.Int32("emit", stat.emit),
		zap.Bool("broken", stat.broken),
		zap.Bool("continued", !stat.noContinued),
		zap.Int32("processed", stat.processed),
		zap.Int32("fwhere", stat.filterWhere),
		zap.Int32("fbwhere", stat.filterBeforeParseWhere),
		zap.Int32("flogparse", stat.filterLogParseError),
		zap.Int32("ftimeparse", stat.filterTimeParseError),
		zap.Int32("fignore", stat.filterIgnore),
		zap.Int32("filterDelay", stat.filterDelay),
		zap.Time("maxDataTime", time.UnixMilli(c.maxDataTimestamp)),
		zap.Time("estimatedMaxDataTime", time.UnixMilli(c.estimatedMaxDataTimestamp)),
	)
}

func (c *Consumer) getCommonTags() (map[string]string, bool) {
	var tags map[string]string
	if c.ct.Target.IsTypePod() {
		if pod := c.getTargetPod(); pod != nil {
			tags = meta.ExtractPodCommonTags(pod.Pod)
		} else {
			return nil, false
		}
	} else {
		tags = meta.ExtractSidecarTags()
	}
	return tags, true
}

func (c *Consumer) addCommonTags(datum []*model.DetailData) bool {
	commonTags, ok := c.getCommonTags()
	if !ok {
		return false
	}

	for _, d := range datum {
		if d.Tags == nil {
			d.Tags = make(map[string]string, len(commonTags))
		}
		for k, v := range commonTags {
			if _, exist := d.Tags[k]; !exist {
				d.Tags[k] = v
			}
		}
	}
	return true
}

func (c *Consumer) SetOutput(output output.Output) {
	c.output = output
}

func (c *Consumer) emit(expectedTs int64) {
	c.updatePeriodStatus(expectedTs, func(status *PeriodStatus) {
		status.estimatedMaxDataTimestamp = c.estimatedMaxDataTimestamp
	})
	c.sub.Emit(expectedTs)
}

// returns true is can continue
func (c *Consumer) executeBeforeParseWhere(ctx *LogContext) bool {
	if c.BeforeParseWhere == nil {
		return true
	}

	// When execute 'beforeParseWhere', we treat whole line group as a string.
	// So many filters don't need to consider the multiline situation.
	if len(ctx.log.Lines) > 1 {
		content := strings.Join(ctx.log.Lines, "\n")
		bak := ctx.log
		ctx.log = &LogGroup{
			Line:  content,
			Lines: []string{content},
		}
		defer func() {
			ctx.log = bak
		}()
	}

	if b, err := c.BeforeParseWhere.Test(ctx); !b {
		if err != nil {
			if ctx.event != nil {
				ctx.event.Error("beforeParseWhere error %+v", err)
			}
			logger.Debugz("[consumer] beforeParseWhere err", zap.String("key", c.key), zap.Error(err))
		} else {
			if ctx.event != nil {
				ctx.event.Info("beforeParseWhere false")
			}
			logger.Debugz("[consumer] filter before where", zap.String("key", c.key), zap.String("line", ctx.log.FirstLine()))
		}
		c.stat.filterBeforeParseWhere++
		return false
	}
	return true
}

func (c *Consumer) executeLogParse(ctx *LogContext) bool {
	if c.LogParser == nil {
		return true
	}
	if err := c.LogParser.Parse(ctx); err != nil {
		logger.Debugz("log parse error", zap.String("consumer", c.key), zap.String("line", ctx.GetLine()), zap.Error(err))
		c.stat.filterLogParseError++
		return false
	}

	return true
}

func (c *Consumer) executeVarsProcess(ctx *LogContext) bool {
	if c.varsProcessor == nil {
		return true
	}
	if vars, err := c.varsProcessor.process(ctx); err != nil {
		logger.Debugz("parse vars error", zap.String("consumer", c.key), zap.String("line", ctx.GetLine()), zap.Error(err))
		c.stat.filterLogParseError++
		return false
	} else {
		ctx.vars = vars
	}

	return true
}

// 返回的是对齐的
func (c *Consumer) executeTimeParse(ctx *LogContext) (int64, bool) {
	ts, err := c.executeTimeParse0(ctx)
	if err != nil {
		logger.Debugz("parse time error", //
			zap.String("consumer", c.key), //
			zap.String("line", ctx.GetLine()),
			zap.Error(err)) //
		if ctx.event != nil {
			ctx.event.Error("parse time error: %+v", err)
		}
		c.stat.filterTimeParseError++
		return 0, false
	}
	return ts, true
}

func (c *Consumer) executeTimeParse0(ctx *LogContext) (int64, error) {
	if c.TimeParser == nil {
		return util.CurrentMS(), nil
	}

	ts, err := c.TimeParser.Parse(ctx)
	if err != nil {
		return 0, err
	}

	if ts <= 0 {
		return 0, errors.New("ts < 0 ")
	}

	return ts, nil
}

func (c *Consumer) executeWhere(ctx *LogContext) bool {
	if c.Where == nil {
		return true
	}
	if b, err := c.Where.Test(ctx); !b {
		if err != nil {
			if ctx.event != nil {
				ctx.event.Error("where error: %+v", err)
			}
			logger.Debugz("[consumer] where err", zap.String("key", c.key), zap.Error(err))
		} else {
			logger.Debugz("[consumer] filter where", zap.String("key", c.key), zap.String("line", ctx.log.FirstLine()))
		}
		c.stat.filterWhere++
		ctx.periodStatus.stat.filterWhere++
		return false
	}

	return true
}

func (c *Consumer) executeSelectAgg(processGroupEvent *event.Event, ctx *LogContext, point *storage.Point) {
	xs := c.Select.(*xSelect)
	// 我们先把所有value select 出来
	var selectedValues map[string]interface{} = nil
	if processGroupEvent != nil {
		selectedValues = make(map[string]interface{}, len(xs.values))
	}
	for j, so := range xs.values {
		// 这里像lego一样有一个where步骤, 这属于很高级的用法了, 很少用到
		if so.where != nil {
			if b, err := so.where.Test(ctx); !b {
				if err != nil {
					if processGroupEvent != nil {
						processGroupEvent.Error("agg where error: %+v", err)
					}
					c.stat.aggWhereError++
					ctx.periodStatus.stat.aggWhereError++
					logger.Debugz("[consumer] agg error", zap.String("consumer", c.key), zap.String("line", ctx.GetLine()), zap.String("as", so.as), zap.Error(err))
				} else {
					logger.Debugz("[consumer] agg where false", zap.String("consumer", c.key), zap.String("line", ctx.GetLine()), zap.String("as", so.as), zap.Error(err))
				}
				continue
			}
		}

		switch so.agg {
		case agg.AggCount:
			if n, ok := point.Values[j].(*storage.AggNumberDataNode); ok {
				n.AddCount()
				if processGroupEvent != nil {
					selectedValues[so.as] = 1
					processGroupEvent.Info("field=[%s], add count, stat.count=[%d]", point.ValueNames[j], n.Count)
				}
			}
		case agg.AggHll:
			str, err := so.elect.ElectString(ctx)
			if err != nil {
				c.stat.selectError++
				ctx.periodStatus.stat.selectError++
				logger.Debugz("[consumer] select string error", zap.String("consumer", c.key), zap.String("line", ctx.GetLine()), zap.String("as", so.as), zap.Error(err))
				continue
			}
			// TODO 此处注意别长期引用这个字符串 否则造成大字符串泄漏
			if n, ok := point.Values[j].(*storage.HllDataNode); ok {
				n.MergeHll(str)
			}
		default:
			// TODO 对于普通聚合型的可能要考虑保留count/sum两个值, 以便能够计算出avg
			number, err := so.elect.ElectNumber(ctx)
			if err != nil {
				if processGroupEvent != nil {
					processGroupEvent.Error("field=[%s], elect number error %+v, skip", point.ValueNames[j], err)
				}
				c.stat.selectError++
				ctx.periodStatus.stat.selectError++
				logger.Debugz("[consumer] select number error", zap.String("consumer", c.key), zap.String("line", ctx.GetLine()), zap.String("as", so.as), zap.Error(err))
				continue
			}

			// ignore NaN Inf
			if math.IsNaN(number) || math.IsInf(number, 0) {
				if processGroupEvent != nil {
					processGroupEvent.Info("field=[%s], number is NaN or Inf, fix to 0", point.ValueNames[j])
				}
				number = 0
			}

			if n, ok := point.Values[j].(*storage.AggNumberDataNode); ok {
				n.MergeNumber(number)
				if processGroupEvent != nil {
					selectedValues[so.as] = number
					processGroupEvent.Info("field=[%s], merge value=[%f], stat.agg=[%+v] stat.count=[%d] stat.value=[%f]", point.ValueNames[j], number, n.Agg, n.Count, n.Value)
				}
			}
		}
	}
	if processGroupEvent != nil {
		processGroupEvent.Set("selectedValues", selectedValues)
	}
}

// get or create storage point, returns nil if creation will exceed maxKeySize
func (c *Consumer) getOrCreateStoragePoint(alignTs int64, ctx *LogContext, shard *storage.Shard, groups []string) *storage.Point {
	pointKey := c.joinPointKey(groups)

	// point 持有这一分钟的聚合结果
	point := shard.GetPoint(pointKey)
	if point == nil {
		if c.maxKeySize > 0 && shard.PointCount() >= c.maxKeySize {
			logger.Debugz("[consumer] filter maxKeySize", zap.String("key", c.key), zap.Int("maxKeySize", c.maxKeySize))
			c.stat.filterGroupMaxKeys++
			ctx.periodStatus.stat.filterGroupMaxKeys++
			return nil
		}
		// 此处 groups 是对 line 的切分引用, 我们不能直接将 groups 保存下来, 否则原始 line 无法释放
		// 因此这里做了一个深复制
		point = c.createPoint(alignTs, util.DeepCopyStringSlice(groups))
		shard.SetPoint(util.DeepCopyString(pointKey), point)
	}

	return point
}

func (c *Consumer) init() {
	if c.sub != nil {
		c.sub.init()
	}
	maxKeySize := c.GroupBy.MaxKeySize()
	if maxKeySize > hardMaxKeySize {
		maxKeySize = hardMaxKeySize
	}
	c.maxKeySize = maxKeySize
}

func (c *Consumer) executeGroupBy(ctx *LogContext) ([]string, bool) {
	groups, err := c.GroupBy.Execute(ctx)
	if err != nil {
		logger.Debugz("[consumer] group error", zap.String("key", c.key), zap.String("line", ctx.log.FirstLine()), zap.Error(err))
		c.stat.filterGroup++
		ctx.periodStatus.stat.filterGroup++
		return nil, false
	}
	return groups, true
}

func removeZeroNumbers(event *pb2.ReportEventRequest_Event) {
	for key, value := range event.Numbers {
		if key != "ok" && value == 0 {
			// this is safe in Golang
			delete(event.Numbers, key)
		}
	}
}
