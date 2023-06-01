/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package batch

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

type (
	batchProcessor struct {
		stopChan   chan struct{}
		buffer     chan interface{}
		bufferSize int

		options *options
		state   int32

		batchMaxWait int64
		batchMaxSize int64

		timer    *time.Timer
		consumer Consumer
	}

	timerSig struct{}
)

const (
	batchProcRunning = iota
	batchProcClosed
)

var (
	bufferChanPool = sync.Pool{
		New: func() interface{} {
			return make(chan interface{}, maxBufferSize)
		},
	}
	maxBufferSize int

	maxSizeLock sync.Mutex

	batchTmpPool = sync.Pool{
		New: func() interface{} {
			return make([]interface{}, 0, maxBatchTmpSize)
		},
	}
	maxBatchTmpSize int
)

func NewBatchProcessor(s int, c Consumer, opts ...OptionFunc) Processor {
	opt := defaultOptions

	for _, o := range opts {
		o(&opt)
	}
	if opt.batchThreshold == 0 || opt.batchThreshold > s {
		opt.batchThreshold = s
	}
	maxSizeLock.Lock()
	if s > maxBufferSize {
		maxBufferSize = s
	}
	if opt.batchThreshold > maxBatchTmpSize {
		maxBatchTmpSize = opt.batchThreshold
	}
	maxSizeLock.Unlock()

	r := &batchProcessor{
		stopChan:   make(chan struct{}, 1),
		buffer:     bufferChanPool.Get().(chan interface{}),
		bufferSize: s,
		options:    &opt,
		state:      batchProcRunning,
		timer:      time.NewTimer(opt.interval),

		batchMaxSize: int64(opt.batchThreshold),
		batchMaxWait: int64(opt.interval / time.Millisecond),

		consumer: c,
	}

	return r
}

func (b *batchProcessor) Put(i interface{}) {
	if b.isClosed() {
		logger.Warnf("[batchProc] shutdown but got a item and force flush")
		b.flush([]interface{}{i})
		return
	}

	b.buffer <- i
}

func (b *batchProcessor) TryPut(i interface{}) bool {
	if b.isClosed() {
		logger.Warnf("[batchProc] shutdown but got a item and force flush")
		b.flush([]interface{}{i})
		return true
	}

	select {
	case b.buffer <- i:
	default:
		logger.Warnf("[batchProc] full buffer!")
		return false
	}
	return true
}

func (b *batchProcessor) Run() {
	go b.notify()
	go b.timerSignal()
}

func (b *batchProcessor) Shutdown() {
	atomic.StoreInt32(&b.state, batchProcClosed)
	b.timer.Stop()
	b.buffer <- nil
	b.stopChan <- struct{}{}
}

func (b *batchProcessor) Num() int {
	return len(b.buffer)
}
func (b *batchProcessor) AdjustBatchMaxSize(i int) {
	logger.Infof("[batchProc] adjust batchMaxSize to %d", i)
	atomic.StoreInt64(&b.batchMaxSize, int64(i))
}

func (b *batchProcessor) AdjustBatchMaxWait(duration int) {
	logger.Infof("[batchProc] adjust batchMaxWait to %ds", duration)
	atomic.StoreInt64(&b.batchMaxWait, int64(duration))
}

func (b *batchProcessor) flush(batch []interface{}) {
	b.timer.Reset(time.Duration(atomic.LoadInt64(&b.batchMaxWait)) * time.Millisecond)
	if len(batch) == 0 {
		return
	}
	b.consumer.Consume(batch)
}

func (b *batchProcessor) notify() {
	batch := batchTmpPool.Get().([]interface{})
	defer func() {
		for i := range batch {
			batch[i] = nil
		}
		batch = batch[:0]
		batchTmpPool.Put(batch)
		bufferChanPool.Put(b.buffer)
		b.buffer = nil
	}()

	var itemsWeight int
	flushFunc := func() {
		b.flush(batch)
		for i := range batch {
			batch[i] = nil
		}
		batch = batch[:0]
		itemsWeight = 0
	}

	for obj := range b.buffer {
		if obj == nil {
			logger.Warnf("[batchProc] buffer receive sig to quit and force flush(size %d)", len(batch))
			flushFunc()
			return
		}
		switch obj.(type) {
		case *timerSig:
			//if logger.IsDebugEnabled() {
			//	logger.Debugf("[batchProc] timeout and force flush(size %d)", len(batch))
			//}
			flushFunc()
		default:
			batch = append(batch, obj)
			// cal weight
			if b.options.calculator != nil {
				itemsWeight += b.options.calculator(obj)
				if itemsWeight >= b.options.weightMax {
					//if logger.IsDebugEnabled() {
					//	logger.Debugf("[batchProc] over weight and flush(size %d, weight %d)", len(batch), itemsWeight)
					//}
					flushFunc()
					continue
				}
			}
			if int64(len(batch)) >= atomic.LoadInt64(&b.batchMaxSize) {
				//if logger.IsDebugEnabled() {
				//	logger.Debugf("[batchProc] size out and flush(size %d)", len(batch))
				//}
				flushFunc()
			}
		}
	}
}

func (b *batchProcessor) timerSignal() {
	for {
		select {
		case <-b.timer.C:
			select {
			case b.buffer <- new(timerSig):
			case <-b.stopChan:
				return
			}
		case <-b.stopChan:
			return
		}
	}
}

func (b *batchProcessor) isClosed() bool {
	return atomic.LoadInt32(&b.state) == batchProcClosed
}
