package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"errors"
	"go.uber.org/zap"
	"runtime"
	"sync"
	"time"
)

const (
	defaultMonitorFileInterval = 10 * time.Second
	// 默认每10s拉一次日志
	defaultPullDelay = 10 * time.Second
	// 日志延迟打印的容忍时间, 也就是我们假设 t+logDelayTolerance 之后, t的日志才会打印到文件里
	logDelayTolerance = 300 * time.Millisecond
)

type (
	listenerImpl struct{}

	// LogPipeline is responsible for detecting log inputs(see inputsManager) , scheduling pulling logs task, and put logs to consumer.
	LogPipeline struct {
		mutex *sync.RWMutex
		st    *api.SubTask

		inputsManager *inputsManager

		// 消费者
		consumer *Consumer

		// 消费者发生更新时候的通知chan
		update chan struct{}
		// 停止的chan
		stop chan struct{}

		// TODO storage 不应该在这里
		s *storage.Storage

		started        bool
		pullWalker     *util.AlignTsWalker
		lastEmitWindow int64
	}
	inputWrapperState int8
)

const (
	inputWrapperStateFirst inputWrapperState = iota
	inputWrapperStateSuccess
	inputWrapperStateError
)

func (iw *inputWrapper) read() (*logstream.ReadResponse, int64, error) {
	return iw.ls.Read(iw.req)
}

func (l *listenerImpl) Changed(path string, ls logstream.LogStream, lcursor int64) {
}

func (p *LogPipeline) SetupConsumer(st *api.SubTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.st = st
	return p.setupConsumer0(st)
}

func (p *LogPipeline) setupConsumer0(st *api.SubTask) (ret error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			logger.Errorf("[pipeline] add consumer [%s]: %v\n%s", st.CT.Config.Key, r, string(buf))
			ret = errors.New("panic error")
		}
	}()

	// 该方法已经跑在锁内
	// 其实我们现在只允许一个pipeline有一个消费者
	logger.Infoz("[pipeline] [log] add consumer", //
		zap.String("key", p.st.CT.Key),                      //
		zap.String("configKey", p.st.CT.Config.Key),         //
		zap.String("configVersion", p.st.CT.Config.Version), //
		zap.String("targetKey", p.st.CT.Target.Key),         //
		zap.String("targetVersion", p.st.CT.Target.Version)) //

	// 论证配置更新时这是原子的
	// 要求:
	// 1. 一般更新时对数据质量没有任何影响, 不会丢采集数据, 不会丢emit
	// 2. 当配置发生不兼容更新时(比如groups增加了), 这显然之前的数据都废了: 1. 能延迟到下一个分钟才生效 或 2.立即生效(之前的数据都不要了)

	// 1. 进入这个方法时整个pipeline是加锁的, 整个pipeline的流程是冻结的, 此时要注意 consumer 的 emitLoop 可能还在跑 (它是在另外一个goroutine里, 有竞争)
	// 2. TODO 创建新的consumer实例, 如果创建失败则返回, 旧的pipeline保持在跑
	// 3. 停止 old consumer 的 emitLoop
	// 4. new consumer 从 old consumer 继承一些状态
	// 5. 启动 new consumer
	// 6. 确保 new consumer emitLoop 是延续上次的进度的 不会丢失 emit

	if p.consumer != nil {
		if p.consumer.ct.Version == st.CT.Version {
			logger.Infoz("[consumer] [log] ignore same version rawTask", //
				zap.String("pipeline", p.st.CT.Key),               //
				zap.String("consumer", st.CT.Key),                 //
				zap.String("configKey", st.CT.Config.Key),         //
				zap.String("configVersion", st.CT.Config.Version), //
			)
			return
		}
	}

	c, err := parseConsumer(st)
	if err != nil {
		return err
	}

	if p.consumer != nil {
		// 继承属性
		c.Update(p.consumer)
		// 彻底停止旧consumer
		p.consumer.Stop()
	}

	if c.timeline == nil {
		c.SetStorage(p.s)
	}

	// 启动新consumer
	if p.started {
		c.Start()
	}
	p.consumer = c

	p.inputsManager.update(st)

	select {
	case p.update <- struct{}{}:
	default:
	}
	return nil
}

func (p *LogPipeline) Update(f func(api.Pipeline)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	util.WithRecover(func() {
		f(p)
	}, func(err interface{}) {
		logger.Errorz("[pipeline] update panic", zap.String("key", p.st.CT.Key), zap.Any("panic", err))
	})
}

func (p *LogPipeline) View(f func(api.Pipeline)) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	f(p)
}

func NewPipeline(st *api.SubTask, s *storage.Storage, lsm *logstream.Manager) (*LogPipeline, error) {
	return &LogPipeline{
		mutex: &sync.RWMutex{},
		st:    st,
		s:     s,
		stop:  make(chan struct{}),
		inputsManager: &inputsManager{
			key:      st.CT.Key,
			listener: &listenerImpl{},
			lsm:      lsm,
			inputs:   make(map[string]*inputWrapper),
		},
		update: make(chan struct{}, 4),
	}, nil
}

func (p *LogPipeline) isStop() bool {
	select {
	case <-p.stop:
		return true
	default:
		return false
	}
}

// 对一个数据源消费一次, 如果该数据源还有更多数据, 就会返回true
func (p *LogPipeline) consumeUntilEndForOneInput(iw *inputWrapper) bool {
	// TODO resp.err 和 err 的关系
	resp, _, err := iw.read()

	if err != nil {
		logger.Errorz("[pipeline] [log] [input] error", zap.String("key", p.st.CT.Key), zap.Error(err))
		return false
	}

	iw.req.Cursor = resp.NextCursor

	if resp.Error == nil {
		iw.state = inputWrapperStateSuccess
	} else {
		iw.state = inputWrapperStateError
	}
	if iw.state != iw.lastState {
		if iw.lastState == inputWrapperStateFirst {
			logger.Infoz("[pipeline] [log] [input] [event] first", //
				zap.String("key", p.st.CT.Key),             //
				zap.String("path", resp.Path),              //
				zap.String("fileId", resp.FileId),          //
				zap.Int64("beginOffset", resp.BeginOffset), //
				zap.Error(resp.Error)) //
		} else {
			logger.Infoz("[pipeline] [log] [input] [event] changed", //
				zap.String("key", p.st.CT.Key),             //
				zap.String("path", resp.Path),              //
				zap.String("fileId", resp.FileId),          //
				zap.Int64("beginOffset", resp.BeginOffset), //
				zap.Error(resp.Error)) //
		}
	}
	iw.lastState = iw.state

	p.consumer.Consume(resp, iw, err)

	return resp.HasMore
}

func (p *LogPipeline) Start() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.setupConsumer0(p.st)

	p.started = true
	if err != nil {
		logger.Errorz("[pipeline] setup consumer error", zap.String("key", p.st.CT.Key), zap.Error(err))
		return
	}

	p.consumer.Start()
	p.startInternalLoop()

	logger.Infoz("[pipeline] start", zap.String("key", p.st.CT.Key))
}

func (p *LogPipeline) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	close(p.stop)

	p.inputsManager.stop()
	p.consumer.Stop()

	logger.Infoz("[pipeline] stop", zap.String("key", p.st.CT.Key))
}

func (p *LogPipeline) printStat() {
	if p.isStop() {
		return
	}
	p.consumer.printStat()
}

func (p *LogPipeline) checkInputLoop() {
	checkInputsTimer := time.NewTicker(defaultMonitorFileInterval)
	defer checkInputsTimer.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-checkInputsTimer.C:
			p.Update(func(pipeline api.Pipeline) {
				// double check
				if p.isStop() {
					return
				}
				p.inputsManager.checkInputsChange()
			})
		}
	}
}

func (p *LogPipeline) printStatLoop() {
	checkInputsTimer := time.NewTicker(time.Minute)
	defer checkInputsTimer.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-checkInputsTimer.C:
			p.View(func(pipeline api.Pipeline) {
				p.printStat()
			})
		}
	}
}

func (p *LogPipeline) pullLoop() {
	// if interval is 5s
	// then we pull logs at [0*5000+300 ms, 1*5000+5300 ms, 2*5000+300 ms, 3*5000+300 ms, ... ]
	// 300ms is for log delay print.
	pullTimer := util.NewAlignTsTimer(p.pullInterval().Milliseconds(), logDelayTolerance.Milliseconds(), 0, 0, false)
	defer pullTimer.Stop()

	pullTimer.Next()
	for {
		select {
		case <-p.stop:
			return
		case <-p.update:
			// update configs, maybe recreate pull timer
			logger.Infof("[pipeline] [log] [%s] update", p.st.CT.Key)

			newPullDelayMs := p.pullInterval().Milliseconds()
			if newPullDelayMs != pullTimer.Interval() {
				// data window changed
				p.lastEmitWindow = 0
				pullTimer.Update(newPullDelayMs, logDelayTolerance.Milliseconds(), 0, 0)
				pullTimer.Next()
			}
		case <-pullTimer.Chan():
			// 多数据源case, 需要遍历所有数据源
			p.Update(func(pipeline api.Pipeline) {
				// double check
				if p.isStop() {
					return
				}

				p.pullAndConsume()
				p.maybeEmit()
			})
			pullTimer.Next()
		}
	}
}

// pipeline 事件循环
func (p *LogPipeline) startInternalLoop() {
	go p.checkInputLoop()
	go p.printStatLoop()
	go p.pullLoop()
}

func (p *LogPipeline) pullInterval() time.Duration {
	interval := defaultPullDelay
	// 对于 1s 和 5s 周期的任务, 拉取频率要适当提高
	if p.consumer.Window.Interval < defaultPullDelay {
		interval = p.consumer.Window.Interval
	}
	return interval
}

// pull logs, consume logs, maybe emit results
func (p *LogPipeline) pullAndConsume() {
	for _, input := range p.inputsManager.inputs {
		// 对于每个 input(file) 拉到完 然后再换下一个
		for p.consumeUntilEndForOneInput(input) {
			// 稍微让一下其他人
			runtime.Gosched()
		}
	}
}

// check last emit time and current data time, maybe trigger an emit.
func (p *LogPipeline) maybeEmit() {
	interval := p.consumer.Window.Interval.Milliseconds()
	lastFinishedWindow := p.consumer.estimatedMaxDataTimestamp/interval*interval - interval
	if lastFinishedWindow <= 0 {
		return
	}
	if p.lastEmitWindow == 0 {
		// p.lastEmitWindow ==0 means lastFinishedWindow is an incomplete window
		// For data accuracy, we discard first incomplete window data.
		p.lastEmitWindow = lastFinishedWindow
	} else if lastFinishedWindow != p.lastEmitWindow {
		// We just emit lastFinishedWindow window data.
		// If our process hang '10 x window' and recover, which is rare case, then we only emit last window data, discarding first 9 window data.
		p.lastEmitWindow = lastFinishedWindow
		p.consumer.emit(lastFinishedWindow)
	}
}
