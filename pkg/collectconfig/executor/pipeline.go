/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"encoding/gob"
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/transfer"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/traas-stack/holoinsight-agent/pkg/util/recoverutils"
	"go.uber.org/zap"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	defaultMonitorFileInterval = 10 * time.Second
	// 默认每10s拉一次日志
	defaultPullDelay = 10 * time.Second
	// Logs generated at T1 may not be printed in the log file until T2.
	// Our system tolerates a delay(T2 - T1) of up to 300.
	// If the delay exceeds this value, it may cause these delayed data to be ignored.
	logDelayTolerance                         = time.Second
	inputWrapperStateFirst inputWrapperStatus = iota
	inputWrapperStateSuccess
	inputWrapperStateError
)

type (
	listenerImpl struct {
		_ int
	}

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

		started bool
		pipelineState
	}
	pipelineState struct {
		pullTimer      *util.AlignedTimer
		lastEmitWindow int64
	}
	inputWrapperStatus int8
	// RunInLock makes a func running in the write lock of the pipeline
	RunInLock        func(func())
	pipelineStateObj struct {
		Inputs         []inputStateObj
		PullTimerState []byte
		ConsumerState  *consumerStateObj
		LastEmitWindow int64
	}
)

func init() {
	gob.Register(&pipelineStateObj{})
}

func (p *LogPipeline) StopAndSaveState(store transfer.StateStore) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.isStopped() {
		return nil
	}

	close(p.stop)

	state := &pipelineStateObj{
		LastEmitWindow: p.lastEmitWindow,
	}
	if pullTimerState, err := p.pullTimer.SaveState(); err != nil {
		return err
	} else {
		state.PullTimerState = pullTimerState
	}

	if s, err := p.consumer.SaveState(); err != nil {
		return err
	} else {
		state.ConsumerState = s
	}

	for _, i := range p.inputsManager.inputs {
		state.Inputs = append(state.Inputs, i.inputStateObj)
	}

	if b, err := util.GobEncode(state); err != nil {
		return err
	} else {
		store.Put(p.st.CT.Key, b)
	}

	return nil
}

func (p *LogPipeline) LoadState(store transfer.StateStore) error {
	i, err := store.Get(p.st.CT.Key)
	if err != nil {
		return err
	}
	state := &pipelineStateObj{}
	if err := util.GobDecode(i.([]byte), state); err != nil {
		return err
	}

	{
		interval := p.pullInterval()
		pullTimer, _ := util.NewAlignedTimer(interval, logDelayTolerance, false, false)
		if err := pullTimer.LoadState(state.PullTimerState); err != nil {
			return err
		}
	}

	if err := p.consumer.LoadState(state.ConsumerState); err != nil {
		return err
	}

	p.lastEmitWindow = state.LastEmitWindow

	for _, s := range state.Inputs {
		ls := p.inputsManager.lsm.AcquireFile(s.Path)
		if err := ls.LoadReadState(s.Cursor); err != nil {
			logger.Infoz("[transfer] [pipeline] log input load state error", zap.String("key", p.Key()), zap.String("path", s.Path), zap.Error(err))
			p.inputsManager.lsm.Release(s.Path, ls)
		} else {
			logger.Infoz("[transfer] [pipeline] log input load state success", zap.String("key", p.Key()), zap.String("path", s.Path))
			_ = ls.AddListener(p.inputsManager.listener)
			p.inputsManager.inputs[s.Path] = &inputWrapper{
				ls:            ls,
				inputStateObj: s,
			}
		}
	}

	return nil
}

func (iw *inputWrapper) read() (*logstream.ReadResponse, int64, error) {
	return iw.ls.Read(iw.Cursor)
}

func (l *listenerImpl) Changed(ls logstream.LogStream, lcursor int64) {
}
func (p *LogPipeline) Key() string {
	return p.st.CT.Key
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

	c.runInLock = func(f func()) {
		p.Update(func(_ api.Pipeline) {
			f()
		})
	}

	// 启动新consumer
	if p.started {
		c.Start()
	}
	p.consumer = c

	if p.started {
		p.inputsManager.update(st)
		select {
		case p.update <- struct{}{}:
		default:
		}
	}
	return nil
}

func (p *LogPipeline) Update(f func(api.Pipeline)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	recoverutils.WithRecover(func() {
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
	p := &LogPipeline{
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
	}

	if err := p.setupConsumer0(st); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *LogPipeline) isStopped() bool {
	select {
	case <-p.stop:
		return true
	default:
		return false
	}
}

// 对一个数据源消费一次, 如果该数据源还有更多数据, 就会返回true
func (p *LogPipeline) consumeUntilEndForOneInput(iw *inputWrapper) bool {
	resp, nextCursor, err := iw.read()
	iw.Cursor = nextCursor

	if err != nil && !os.IsNotExist(err) {
		logger.Errorz("[pipeline] [log] [input] read error", zap.String("key", p.st.CT.Key), zap.Error(err))
	}

	if resp == nil {
		return false
	}

	if err == nil {
		iw.State = inputWrapperStateSuccess
	} else {
		iw.State = inputWrapperStateError
	}
	if iw.State != iw.LastState {
		if iw.LastState == inputWrapperStateFirst {
			logger.Infoz("[pipeline] [log] [input] [event] first", //
				zap.String("key", p.st.CT.Key),  //
				zap.String("path", resp.Path),   //
				zap.String("range", resp.Range), //tail -1111 in
				zap.Error(err))                  //
		} else {
			logger.Infoz("[pipeline] [log] [input] [event] changed", //
				zap.String("key", p.st.CT.Key),  //
				zap.String("path", resp.Path),   //
				zap.String("range", resp.Range), //
				zap.Error(err))                  //
		}
	}
	iw.LastState = iw.State

	p.consumer.Consume(resp, iw, err)

	return resp.HasMore
}

func (p *LogPipeline) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.inputsManager.update(p.st)
	p.consumer.Start()
	p.startInternalLoop()
	p.started = true
	logger.Infoz("[pipeline] start", zap.String("key", p.st.CT.Key))
	return nil
}

func (p *LogPipeline) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.isStopped() {
		return
	}

	close(p.stop)
	// TODO wait until xxxLoop exited

	p.inputsManager.stop()
	p.consumer.Stop()

	logger.Infoz("[pipeline] stop", zap.String("key", p.st.CT.Key))
}

func (p *LogPipeline) printStat() {
	if p.isStopped() {
		return
	}
	p.consumer.printStat()
	p.inputsManager.ld.touch()
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
				if p.isStopped() {
					return
				}
				p.inputsManager.checkInputsChange()
			})
		}
	}
}

func (p *LogPipeline) pullLoop() {
	// if interval is 5s
	// then we pull logs at [0*5000+300 ms, 1*5000+5300 ms, 2*5000+300 ms, 3*5000+300 ms, ... ]
	// 300ms is for log delay print.

	interval := p.pullInterval()

	pullTimer := p.pullTimer
	if pullTimer == nil {
		pullTimer, _ = util.NewAlignedTimer(interval, logDelayTolerance, false, false)
		p.pullTimer = pullTimer
	}
	defer pullTimer.Stop()
	lastPrintStatTime := pullTimer.NextEmitTime().Truncate(time.Minute).Add(logDelayTolerance)

	// trigger pull right now
	first := make(chan struct{}, 1)
	first <- struct{}{}

	for {
		select {
		case <-p.stop:
			return
		case <-p.update:
			// update configs, maybe recreate pull timer
			logger.Infof("[pipeline] [log] [%s] update", p.st.CT.Key)

			newInterval := p.pullInterval()
			if interval != newInterval {
				pullTimer.Stop()

				interval = p.pullInterval()
				pullTimer, _ = util.NewAlignedTimer(interval, logDelayTolerance, false, false)
				p.pullTimer = pullTimer

				// data window changed
				p.lastEmitWindow = 0
			}
		case <-first:
			p.Update(func(pipeline api.Pipeline) {
				if p.isStopped() {
					return
				}

				p.pullAndConsume()
				p.maybeEmit()
			})
		case <-pullTimer.C:
			p.Update(func(pipeline api.Pipeline) {
				if p.isStopped() {
					return
				}

				p.pullAndConsume()
				p.maybeEmit()

				// Print stat on the first execution of every minute
				nextEmitTime := pullTimer.NextEmitTime()
				if pullTimer.NextEmitTime().After(lastPrintStatTime) && lastPrintStatTime.Minute() != nextEmitTime.Minute() {
					lastPrintStatTime = nextEmitTime
					p.printStat()
				}

				pullTimer.Next()
			})
		}
	}
}

// pipeline 事件循环
func (p *LogPipeline) startInternalLoop() {
	go p.checkInputLoop()
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

// pull logs, and then consume logs
func (p *LogPipeline) pullAndConsume() {
	if len(p.inputsManager.inputs) == 0 {
		p.consumer.stat.Miss = true
		return
	}

	for _, input := range p.inputsManager.inputs {
		for p.consumeUntilEndForOneInput(input) {
			// give up scheduling
			runtime.Gosched()
		}
	}
}

// check last emit time and current data time, maybe trigger an emit.
func (p *LogPipeline) maybeEmit() {
	interval := p.consumer.Window.Interval.Milliseconds()

	lastFinishedWindow := p.consumer.watermark/interval*interval - interval

	if lastFinishedWindow <= 0 {
		return
	}
	if p.lastEmitWindow == 0 {
		// p.lastEmitWindow ==0 means lastFinishedWindow is an incomplete window
		// For data accuracy, we discard first incomplete window data.
		p.lastEmitWindow = lastFinishedWindow
	} else if lastFinishedWindow != p.lastEmitWindow {
		for ts := lastFinishedWindow; ts > p.lastEmitWindow; ts -= interval {
			p.consumer.emit(ts)
		}
		p.lastEmitWindow = lastFinishedWindow
	}
}
