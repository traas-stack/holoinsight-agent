/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import (
	"sync"
	"time"
)

type (
	// Manager 用于确保相同路径的path只有一个 FileLogStream 实例
	Manager struct {
		mutex          sync.Mutex
		cache          map[string]*cacheItem
		maxLineSize    int
		maxIOReadBytes int64
		stop           chan struct{}
	}
	cacheItem struct {
		ref int32
		ls  LogStream
	}
)

func NewManager() *Manager {
	return NewManager2(DefaultLogInputConfig.MaxLineSize, DefaultLogInputConfig.MaxIOReadBytes)
}

func NewManager2(maxLineSize int, maxIOReadBytes int64) *Manager {
	return &Manager{
		cache:          make(map[string]*cacheItem),
		maxLineSize:    maxLineSize,
		maxIOReadBytes: maxIOReadBytes,
		stop:           make(chan struct{}),
	}
}

func (m *Manager) Start() {
	go m.cleanLoop()
}

func (m *Manager) cleanLoop() {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-timer.C:
			m.cleanOnce()
			timer.Reset(time.Minute)
		}
	}
}

func (m *Manager) cleanOnce() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, item := range m.cache {
		item.ls.Clean()
	}
}

func (m *Manager) Acquire(path string) LogStream {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	i := m.cache[path]
	if i != nil {
		i.ref++
		return i.ls
	}

	ls := NewFileLogStream(LogInputConfig{
		Path:           path,
		MaxLineSize:    m.maxLineSize,
		MaxIOReadBytes: m.maxIOReadBytes,
	})
	ls.Start()
	i = &cacheItem{
		ref: 1,
		ls:  ls,
	}
	m.cache[path] = i
	return i.ls
}

func (m *Manager) Release(path string, ls LogStream) {
	m.mutex.Lock()
	i := m.cache[path]
	i.ref--
	if i.ref == 0 {
		delete(m.cache, path)
		ls.Stop()
	}
	m.mutex.Unlock()
}
