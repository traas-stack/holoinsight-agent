/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import "container/list"

const (
	modeAcquire = iota
	modeRelease
	defaultChCap = 4096
)

type (
	memoryLimiter struct {
		totalBytes int64
		used       int64
		ch         chan *task
		pending    *list.List
	}
	task struct {
		mode  int
		bytes int64
		ok    chan struct{}
	}
)

func newMemoryLimiter(totalBytes int64) *memoryLimiter {
	return &memoryLimiter{
		totalBytes: totalBytes,
		used:       0,
		ch:         make(chan *task, defaultChCap),
		pending:    list.New(),
	}
}

func (ml *memoryLimiter) acquire(bytes int64) {
	if bytes == 0 {
		return
	}
	t := &task{
		mode:  modeAcquire,
		bytes: bytes,
		ok:    make(chan struct{}),
	}
	ml.ch <- t
	<-t.ok
}

func (ml *memoryLimiter) release(bytes int64) {
	if bytes == 0 {
		return
	}
	t := &task{
		mode:  modeRelease,
		bytes: bytes,
		ok:    nil,
	}
	ml.ch <- t
}

func (ml *memoryLimiter) start() {
	go ml.loop()
}

func (ml *memoryLimiter) loop() {
	for {
		t := <-ml.ch
		switch t.mode {
		case modeAcquire:
			if !ml.acquire0(t) {
				ml.pending.PushBack(t)
			}
		case modeRelease:
			ml.used -= t.bytes
			for ml.pending.Len() > 0 {
				p := ml.pending.Front()
				if ml.acquire0(p.Value.(*task)) {
					ml.pending.Remove(p)
				} else {
					break
				}
			}
		}
	}
}

func (ml *memoryLimiter) acquire0(t *task) bool {
	if ml.used+t.bytes <= ml.totalBytes {
		ml.used += t.bytes
		if ml.ch != nil {
			close(ml.ch)
		}
		return true
	}
	return false
}
