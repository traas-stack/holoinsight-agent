/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/transfer"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"time"
)

type (
	GLogStream struct {
		Key       string
		Mutex     sync.RWMutex
		Cache     sync.Map
		Listeners []Listener

		Cursor       int64
		PendingBytes int64
		PendingReads int32

		sub SubLogStream
	}

	SubLogStream interface {
		transfer.StatefulInput
		Start()
		Stop()
		Read(resp *ReadResponse) error
		CreateResponse(cursor int64) *ReadResponse
	}
)

var (
	_ LogStream = &GLogStream{}
)

func (f *GLogStream) Start() {
	f.sub.Start()
}

func (f *GLogStream) Stop() {
	f.sub.Stop()
}

func (f *GLogStream) UpdatePending(resp *ReadResponse, add bool) {
	if add {
		atomic.AddInt64(&f.PendingBytes, resp.Bytes)
		atomic.AddInt32(&f.PendingReads, 1)
	} else {
		atomic.AddInt64(&f.PendingBytes, -resp.Bytes)
		atomic.AddInt32(&f.PendingReads, -1)
	}
}

func (f *GLogStream) getFromCache(cursor int64) *ReadResponse {
	if c, ok := f.Cache.Load(cursor); ok {
		cached := c.(*cachedRead)
		resp := cached.resp

		if atomic.AddInt32(&cached.pendingReads, -1) == 0 {
			f.Cache.Delete(cursor)
			f.UpdatePending(resp, false)
		}

		if !resp.HasMore {
			fc := atomic.LoadInt64(&f.Cursor)
			if resp.Cursor+1 < fc {
				resp.HasMore = true
			}
		}

		return resp
	} else {
		return nil
	}
}

func (f *GLogStream) Read(reqCursor int64) (*ReadResponse, int64, error) {
	// check cache
	if c := f.getFromCache(reqCursor); c != nil {
		return c, c.Cursor + 1, c.error
	}

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	// double check cache
	if c := f.getFromCache(reqCursor); c != nil {
		return c, c.Cursor + 1, c.error
	}

	if reqCursor != f.Cursor {
		return nil, f.Cursor, fmt.Errorf("stale cursor request=[%d] current=[%d]", reqCursor, f.Cursor)
	}

	resp := f.sub.CreateResponse(f.Cursor)
	if resp.IOStartTime.IsZero() {
		resp.IOStartTime = time.Now()
	}
	err := f.sub.Read(resp)
	if resp.IOEndTime.IsZero() {
		resp.IOEndTime = time.Now()
	}
	resp.error = err

	// save to cache
	if len(f.Listeners) > 1 {
		f.UpdatePending(resp, true)
		f.Cache.Store(resp.Cursor, &cachedRead{
			pendingReads: int32(len(f.Listeners) - 1),
			resp:         resp,
		})
	}

	atomic.StoreInt64(&f.Cursor, f.Cursor+1)

	f.fire(f.Cursor - 1)

	return resp, resp.Cursor + 1, resp.error
}

func (f *GLogStream) fire(cursor int64) {
	listeners := f.Listeners
	for _, l := range listeners {
		l.Changed(f, cursor)
	}
}

func (f *GLogStream) AddListener(listener Listener) int64 {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	f.Listeners = append(f.Listeners, listener)
	return f.Cursor
}

func (f *GLogStream) RemoveListener(listener Listener, cursor int64) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	listeners := make([]Listener, 0, len(f.Listeners))
	for _, l := range f.Listeners {
		// The implementations of Listener must not be empty structs.
		// They will result in same object address when their instances are converted to Listener interface,
		// which leads to the following 'l!=listener' always returns false
		if l != listener {
			listeners = append(listeners, l)
		}
	}
	if x := len(f.Listeners) - len(listeners); x > 1 {
		logger.Errorz("remove multi listeners", zap.String("key", f.Key), zap.Int("old", len(f.Listeners)), zap.Int("new", len(listeners)))
	}
	f.Listeners = listeners

	for i := cursor; i < f.Cursor; i++ {
		f.getFromCache(i)
	}
}

func (f *GLogStream) Clean() {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	expiredTime := time.Now().Add(-expireTimeout)
	f.Cache.Range(func(key, value interface{}) bool {
		cached := value.(*cachedRead)
		resp := cached.resp
		if resp.IOStartTime.Before(expiredTime) {
			f.Cache.Delete(key)
			f.UpdatePending(resp, false)
			logger.Errorz("clean cursor cache", //
				zap.String("key", f.Key),                       //
				zap.Int64("cursor", resp.Cursor),               //
				zap.Int32("pendingReads", cached.pendingReads), //
				zap.Time("ioStartTime", resp.IOStartTime))      //
		}
		return true
	})
}

func (f *GLogStream) Stat() Stat {
	return Stat{
		LatestCursor: atomic.LoadInt64(&f.Cursor),
		PendingBytes: atomic.LoadInt64(&f.PendingBytes),
		PendingReads: atomic.LoadInt32(&f.PendingReads),
	}
}

func (f *GLogStream) LoadReadState(cursor int64) error {
	if cursor > f.Cursor {
		return fmt.Errorf("invalid cursor now=[%d] requested=[%d]", f.Cursor, cursor)
	}

	for index := cursor; index < f.Cursor; index++ {
		cached, ok := f.Cache.Load(index)
		if !ok {
			return fmt.Errorf("no cache for cursor now=[%d] requested=[%d]", f.Cursor, cursor)
		}
		cached.(*cachedRead).pendingReads++
	}

	return nil
}

func (f *GLogStream) SaveState() (interface{}, error) {
	return f.sub.SaveState()
}

func (f *GLogStream) LoadState(i interface{}) error {
	return f.sub.LoadState(i)
}
