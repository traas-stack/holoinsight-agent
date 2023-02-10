package sys

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	"runtime"
	"sync"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
)

type (
	// TODO 这个命名太狭隘
	// 这个pipeline起码可以用于所有 简单的数据源(比如各种系统指标)
	// 然后对数据做各种transform, 然后将数据push
	SysPipeline struct {
		ct       *collecttask.CollectTask
		sqlTask  *collectconfig.SQLTask
		in       input.Input
		interval time.Duration
		mutex    sync.RWMutex
		c        *Consumer
		status   int32
		// 事先把tags算好
		tags map[string]string
	}
)

func (p *SysPipeline) Update(f func(api.Pipeline)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	f(p)
}

func (p *SysPipeline) View(f func(api.Pipeline)) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	f(p)
}

func (p *SysPipeline) SetupConsumer(st *api.SubTask) (retError error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			logger.Errorf("[pipeline] [sys] add consumer error [%s]: %v\n%s", st.CT.Config.Key, r, string(buf))
			retError = errors.New("panic error")
		}
	}()

	logger.Infoz("[pipeline] [sys] add consumer", //
		zap.String("configKey", st.CT.Config.Key),         //
		zap.String("configVersion", st.CT.Config.Version), //
		zap.String("targetKey", st.CT.Target.Version),     //
		zap.String("targetVersion", st.CT.Target.Version), //
	) //

	c, err := ParseConsumer(st.CT, st.SqlTask)
	if err != nil {
		return err
	}
	old := p.c
	if old != nil {
		old.Stop()
	}
	p.c = c
	return nil
}

// 对于 sys 先不考虑合并问题了
func NewSysPipeline(ct *collecttask.CollectTask, sqlTask *collectconfig.SQLTask) (*SysPipeline, error) {
	if sqlTask.From == nil {
		return nil, errors.New("from is nil")
	}
	if sqlTask.ExecuteRule == nil {
		return nil, errors.New("executeRule is nil")
	}
	interval, err := util.ParseDuration(sqlTask.ExecuteRule.FixedRate)
	if err != nil {
		return nil, err
	}

	// TODO 类型不能改, 这个配置原本是cpu的, 不能突然改成mem了.
	in, err := input.Parse(sqlTask.From.Type, sqlTask.From)
	if err != nil {
		return nil, err
	}

	var tags map[string]string = nil

	if sqlTask.GroupBy != nil && len(sqlTask.GroupBy.Groups) > 0 {
		tags = make(map[string]string, len(sqlTask.GroupBy.Groups))
		for _, g := range sqlTask.GroupBy.Groups {
			if g.Elect != nil && g.Elect.Type == "refMeta" && g.Elect.RefMeta != nil {
				value := ""
				switch g.Elect.RefMeta.Name {
				case "ip":
					value = util.GetLocalIp()
				case "hostname":
					value = util.GetHostname()
				default:
					// nothing
				}
				if value != "" {
					tags[g.Name] = value
				}
			}
		}
	}

	if _, ok := tags["app"]; !ok {
		tags["app"] = appconfig.StdAgentConfig.App
	}

	return &SysPipeline{
		ct:       ct,
		in:       in,
		sqlTask:  sqlTask,
		interval: interval,
		tags:     tags,
	}, nil
}

func (p *SysPipeline) Start() {
	logger.Infoz("[pipeline] [sys] start", //
		zap.String("key", p.ct.Key), //
		zap.Any("sqlTask", p.sqlTask),
		zap.Duration("interval", p.interval)) //

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 这里几乎就是走拉模式
	// time.NewTimer(time)
	// 一般都是 fixedRate 模式, 记得对齐
	p.in.Start()
	p.c.Start()

	go p.emitLoop()
}

func (p *SysPipeline) emitOnce(alignTs int64) {
	// TODO context 里写啥呢? 要写执行interval 这样source才能正确计算每秒的值
	// 因为我们很多单位是 xxx/s, 比如bytes/s
	array, err := p.in.Collect(&input.CollectContext{})
	if err != nil {
		logger.Infoz("collect error", zap.String("input", p.sqlTask.From.Type), zap.Error(err))
	} else {
		if len(array) == 0 {
			return
		}
		for _, d := range array {
			d.Timestamp = alignTs
			p.addCommonTags(d)
		}
		p.c.Consume(array)
	}
}

func (p *SysPipeline) emitLoop() {
	// TODO p.interval 可能会变化

	intervalMs := p.interval.Milliseconds()
	w := util.NewAlignTsWalker(intervalMs, 0, 0, 0, false)

	nextTs, delay := w.Next()
	timer := time.NewTimer(delay)
	defer timer.Stop()

	// TODO 将align的算法搬到这里来
	for {
		select {
		// 这个时间是对齐的
		case <-timer.C:
			ret := false
			func() {
				p.mutex.Lock()
				defer p.mutex.Unlock()
				// 已经stop了
				if p.status == 1 {
					ret = true
					return
				}
				alignTs := nextTs - intervalMs
				logger.Infoz("[pipeline] [sys] emit", //
					zap.String("key", p.ct.Key),             //
					zap.String("type", p.sqlTask.From.Type), //
					zap.Int64("ts", alignTs),                //
				)
				util.WithRecover(func() {
					p.emitOnce(alignTs)
				})
			}()
			if ret {
				return
			}
			nextTs, delay = w.Next()
			if delay == 0 {
				runtime.Gosched()
			}
			timer.Reset(delay)
		}
	}
}

func (p *SysPipeline) addCommonTags(d *model.DetailData) {
	if p.tags != nil {
		if d.Tags != nil {
			for k, v := range p.tags {
				d.Tags[k] = v
			}
		} else {
			d.Tags = p.tags
		}
		return
	}
	if d.Tags == nil {
		d.Tags = make(map[string]string)
	}
	// TODO 补充常见tags, 做成通用的

	// 对于非sidecar场景, 比如从 prometheus 获取 则不需要补充元数据

	if _, ok := d.Tags["ip"]; !ok {
		// TODO ip的获取
		// 对于sidecar场景就是取本机ip
		d.Tags["ip"] = util.GetLocalIp()
	}
	if _, ok := d.Tags["host"]; !ok {
		if _, ok := d.Tags["hostname"]; !ok {
			d.Tags["host"] = util.GetHostname()
		}
	}
	if _, ok := d.Tags["app"]; !ok {
		// TODO app 的获取
		d.Tags["app"] = "test"
	}
}

func (p *SysPipeline) Stop() {
	logger.Infoz("[pipeline] stop", zap.String("key", p.ct.Key))
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.status == 0 {
		p.status = 1
		p.in.Stop()
		if p.c != nil {
			p.c.Stop()
		}
	}
}

// TODO 废弃
// UpdateFromOld 发生一次配置update
func (p *SysPipeline) UpdateFromOld(old *SysPipeline) {
	logger.Infoz("[pipeline] update",
		zap.String("key", p.ct.Key),
		// TODO 打印版本号
	)
	// 可以从 old 上继承一些东西, 从而保证配置平滑
	// old.Stop()
	// p.Start()
}
