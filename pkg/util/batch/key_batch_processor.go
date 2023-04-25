/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package batch

import (
	"sync/atomic"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

type (
	batchProcessorWithKey struct {
		stopChan   chan struct{}
		buffer     chan interface{}
		bufferSize int
		keyNum     uint64

		options      *options
		batchMaxWait int64
		batchMaxSize int64

		timer    *time.Timer
		consumer KeyConsumer
	}

	withKey struct {
		key  string
		data interface{}
	}

	withWeight struct {
		itemsWeight int
		data        []interface{}
	}
)

func NewBatchProcessorWithKey(s int, c KeyConsumer, opts ...OptionFunc) KeyBatchProcessor {
	opt := defaultOptions

	for _, o := range opts {
		o(&opt)
	}
	if opt.batchThreshold == 0 || opt.batchThreshold > s {
		opt.batchThreshold = s
	}

	r := &batchProcessorWithKey{
		stopChan:     make(chan struct{}, 1),
		buffer:       make(chan interface{}, s),
		bufferSize:   s,
		options:      &opt,
		batchMaxSize: int64(opt.batchThreshold),
		batchMaxWait: int64(opt.interval / time.Second),
		timer:        time.NewTimer(opt.interval),

		consumer: c,
	}

	return r
}

func (b *batchProcessorWithKey) TryPut(s string, i interface{}) bool {
	select {
	case b.buffer <- &withKey{key: s, data: i}:
	default:
		logger.Warnf("[batchProc] full buffer!")
		return false
	}
	return true
}

func (b *batchProcessorWithKey) Run() {
	go b.notify()
	go b.timerSignal()
}

func (b *batchProcessorWithKey) KeyNum() int {
	return int(atomic.LoadUint64(&b.keyNum))
}

func (b *batchProcessorWithKey) BufferNum() int {
	return len(b.buffer)
}

func (b *batchProcessorWithKey) Shutdown() {
	b.timer.Stop()
	b.buffer <- nil
	b.stopChan <- struct{}{}
}

func (b *batchProcessorWithKey) AdjustBatchMaxSize(i int) {
	logger.Infof("[batchProc] adjust batchMaxSize to %d", i)
	atomic.StoreInt64(&b.batchMaxSize, int64(i))
}

func (b *batchProcessorWithKey) AdjustBatchMaxWait(duration int) {
	logger.Infof("[batchProc] adjust batchMaxWait to %ds", duration)
	atomic.StoreInt64(&b.batchMaxWait, int64(duration))
}

func (b *batchProcessorWithKey) flush(key string, batch []interface{}) {
	if len(batch) == 0 {
		return
	}
	b.consumer.Consume(key, batch)
}

func (b *batchProcessorWithKey) notify() {
	batchWithKey := make(map[string]*withWeight)

	flush := func(k string, v *withWeight) {
		b.flush(k, v.data)
		v.data = v.data[:0]
		v.itemsWeight = 0
	}

	flushOnce := func(k string, v *withWeight) {
		//b.timer.Reset(time.Duration(atomic.LoadInt64(&b.batchMaxWait)) * time.Second)
		flush(k, v)
	}

	flushAll := func() {
		b.timer.Reset(time.Duration(atomic.LoadInt64(&b.batchMaxWait)) * time.Second)
		for k, v := range batchWithKey {
			flush(k, v)
		}
	}

	for obj := range b.buffer {
		if obj == nil {
			if logger.IsDebugEnabled() {
				xx, yy := getSize(batchWithKey)
				logger.Warnf("[batchProc] buffer receive sig to quit and force flush(keyN %d, size {%+v, %d})", len(batchWithKey), xx, yy)
			}
			flushAll()
			return
		}
		switch x := obj.(type) {
		case *timerSig:
			if logger.IsDebugEnabled() {
				xx, yy := getSize(batchWithKey)
				logger.Debugf("[batchProc] timeout and force flush(keyN %d, size {%+v, %d})", len(batchWithKey), xx, yy)
			}
			flushAll()
			batchWithKey = b.cleanKey(batchWithKey)
		case *withKey:
			// new key
			batch, exist := batchWithKey[x.key]
			if !exist {
				batch = &withWeight{
					data: make([]interface{}, 0), // todo: 池化？
				}
				batchWithKey[x.key] = batch
				atomic.AddUint64(&b.keyNum, 1)
			}
			batch.data = append(batch.data, x.data)

			// cal weight
			if b.options.calculator != nil {
				batch.itemsWeight += b.options.calculator(x.data)
				if batch.itemsWeight >= b.options.weightMax {
					logger.Debugf("[batchProc] over weight and flush(size %d, weight %d)", len(batch.data), batch.itemsWeight)
					flushOnce(x.key, batch)
					continue
				}
			}
			if int64(len(batch.data)) >= atomic.LoadInt64(&b.batchMaxSize) {
				logger.Debugf("[batchProc] size out and flush(size %d, weight %d)", len(batch.data), batch.itemsWeight)
				flushOnce(x.key, batch)
			}
		default:
			logger.Errorf("[batchProc] unknown obj type: %T", obj)
		}
	}
}

func (b *batchProcessorWithKey) timerSignal() {
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

func (b *batchProcessorWithKey) cleanKey(m map[string]*withWeight) map[string]*withWeight {
	for k, v := range m {
		if len(v.data) == 0 {
			logger.Debugf("[batchProc] clean key: %s", k)
			delete(m, k)
			atomic.AddUint64(&b.keyNum, ^uint64(0))
		}
	}
	ret := m
	if len(m) == 0 {
		ret = make(map[string]*withWeight)
	}

	return ret
}

func getSize(m map[string]*withWeight) ([]int, int) {
	var ret []int
	for _, v := range m {
		ret = append(ret, len(v.data))
	}
	s := 0
	for i := range ret {
		s += ret[i]
	}
	return ret, s
}
