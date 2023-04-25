/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"time"
)

type (
	// 我们期望在 [offset, offset+interval, offset+2*interval, ... ] 时间点做一些事情
	// 并且要能正确处理时间扭曲的情况, 比如墙上时间为 offset+interval+1ms 时, 由于"某种原因"流程得不到调度
	// 等到了 offset+interval+1000s 后才得到调度, 那么要能正确弥补这期间发生的若干个时间的事件
	// 为了避免热点, 还要能适当打散, 比如 offset+interval 再稍微施加一个小的随机延迟
	AlignTsWalker struct {
		Interval        int64
		offset          int64
		min             int64
		max             int64
		nextExecuteTime int64
		skipFirstTs     bool
	}
	AlignTsTimer struct {
		w     *AlignTsWalker
		timer *time.Timer
	}
)

func NewAlignTsTimer(interval, offset, min, max int64, skipFirstTs bool) *AlignTsTimer {
	timer := time.NewTimer(0)
	select {
	case <-timer.C:
	}
	timer.Stop()
	return &AlignTsTimer{
		w:     NewAlignTsWalker(interval, offset, min, max, skipFirstTs),
		timer: timer,
	}
}

func (t *AlignTsTimer) Chan() <-chan time.Time {
	return t.timer.C
}

func (t *AlignTsTimer) Next() int64 {
	ts, delay := t.w.Next()
	t.timer.Reset(delay)
	return ts
}

func (t *AlignTsTimer) Stop() {
	t.timer.Stop()
}

func (t *AlignTsTimer) Update(interval int64, i int64, min int64, max int64) {
	t.w = NewAlignTsWalker(interval, i, min, max, false)
}

func (t *AlignTsTimer) Interval() int64 {
	return t.w.Interval
}

func NewAlignTsWalker(interval, offset, min, max int64, skipFirstTs bool) *AlignTsWalker {
	return &AlignTsWalker{
		Interval:    interval,
		offset:      offset,
		min:         min,
		max:         max,
		skipFirstTs: skipFirstTs,
	}
}

func NewAlignTsWalker2(interval time.Duration, offset, min, max int64) *AlignTsWalker {
	return &AlignTsWalker{
		Interval: int64(interval / time.Millisecond),
		offset:   offset,
		min:      min,
		max:      max,
	}
}

func (w *AlignTsWalker) Next() (int64, time.Duration) {
	now := CurrentMS()

	// ==0 意味着这是第一次emit
	if w.nextExecuteTime == 0 {
		w.nextExecuteTime = now/w.Interval*w.Interval + w.Interval
		if w.skipFirstTs {
			// 当前这周期肯定是不完整的, 故再加一个interval
			// 这有个缺点会导致第一个周期一定没数据
			w.nextExecuteTime += w.Interval
		}
		firstDelay := w.nextExecuteTime + w.offset + RandBetween64(w.min, w.max) - now
		return w.nextExecuteTime, time.Duration(firstDelay) * time.Millisecond
	} else {
		w.nextExecuteTime += w.Interval
		// 下一次期望在 [maxReadyAlignTs+secondEmitDelayMin, maxReadyAlignTs+secondEmitDelayMax) 期间执行一次
		// c.lastAlignTs 记录着下一个需要发射的周期
		nextExecuteTime := w.nextExecuteTime + w.offset + RandBetween64(w.min, w.max)
		if nextExecuteTime > now {
			return w.nextExecuteTime, time.Duration(nextExecuteTime-now) * time.Millisecond
		} else {
			return w.nextExecuteTime, 0
		}
	}
}
