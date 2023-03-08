package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/dryrun/event"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"math"
	"os"
	"strings"
	"time"
)

const (
	hardMaxKeySize = 100_000
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
		hasLogDelay            bool
		emit                   int32
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
)

func (c *Consumer) AddBatchDetailDatus(datum []*model.DetailData) {
	for _, d := range datum {
		c.addCommonTags(d)
	}
	c.output.WriteBatchAsync(c.ct.Config.Key, c.ct.Target.Key, c.metricName, datum)
}

func (c *Consumer) Consume(resp *logstream.ReadResponse, iw *inputWrapper, err error) {
	c.stat.ioTotal++
	if err != nil {
		// 丢弃
		// c.singleLine.reset()
		c.stat.ioError++
		logger.Errorz("[consumer] [log] digest",
			zap.String("key", c.key),
			zap.Any("resp", resp),
			zap.Error(err))
		return
	}
	fileNotExists := os.IsNotExist(resp.Error)
	if fileNotExists {
		c.stat.miss = true
		logger.Infoz("[consumer] [log] digest, file not exist", //
			zap.String("key", c.key),    //
			zap.String("path", iw.path), //
		)
		return
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

func (c *Consumer) printStat() {
	stat := c.stat
	c.stat = consumerStat{}

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
		zap.Bool("hasLogDelay", stat.hasLogDelay),
		zap.Bool("miss", stat.miss),
		zap.Time("maxDataTime", time.UnixMilli(c.maxDataTimestamp)),
		zap.Time("estimatedMaxDataTime", time.UnixMilli(c.estimatedMaxDataTimestamp)),
	)
}

func (c *Consumer) addCommonTags(d *model.DetailData) {
	if d.Tags == nil {
		d.Tags = make(map[string]string)
	}
	if c.ct.Target.IsTypePod() {
		if _, ok := d.Tags["ip"]; !ok {
			d.Tags["ip"] = c.ct.Target.GetIP()
		}
		if _, ok := d.Tags["hostname"]; !ok {
			d.Tags["hostname"] = c.ct.Target.GetHostname()
		}
		if _, ok := d.Tags["app"]; !ok {
			d.Tags["app"] = c.ct.Target.GetApp()
		}
		if _, ok := d.Tags["namespace"]; !ok {
			d.Tags["namespace"] = c.ct.Target.GetNamespace()
		}
		if _, ok := d.Tags["pod"]; !ok {
			d.Tags["pod"] = c.ct.Target.GetPodName()
		}
	} else {
		if _, ok := d.Tags["ip"]; !ok {
			d.Tags["ip"] = util.GetLocalIp()
		}
		if _, ok := d.Tags["host"]; !ok {
			d.Tags["host"] = util.GetHostname()
		}
		if _, ok := d.Tags["hostname"]; !ok {
			d.Tags["hostname"] = util.GetHostname()
		}
		if _, ok := d.Tags["app"]; !ok && appconfig.StdAgentConfig.App != "" {
			d.Tags["app"] = appconfig.StdAgentConfig.App
		}
	}
}

func (c *Consumer) SetOutput(output output.Output) {
	c.output = output
}

func (c *Consumer) emit(expectedTs int64) {
	c.sub.Emit(expectedTs)
}

// returns true is can continue
func (c *Consumer) executeBeforeParseWhere(ctx *LogContext) bool {
	if c.BeforeParseWhere == nil {
		return true
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
func (c *Consumer) getOrCreateStoragePoint(alignTs int64, shard *storage.Shard, groups []string) *storage.Point {
	pointKey := c.joinPointKey(groups)

	// point 持有这一分钟的聚合结果
	point := shard.GetPoint(pointKey)
	if point == nil {
		if c.maxKeySize > 0 && shard.PointCount() >= c.maxKeySize {
			logger.Debugz("[consumer] filter maxKeySize", zap.String("key", c.key), zap.Int("maxKeySize", c.maxKeySize))
			c.stat.filterGroupMaxKeys++
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
		return nil, false
	}
	return groups, true
}
