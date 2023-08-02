/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/transfer"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

const (
	stateKey = "LogStream.Manager"
)

type (
	// Manager 用于确保相同路径的path只有一个 FileLogStream 实例
	Manager struct {
		mutex          sync.Mutex
		maxLineSize    int
		maxIOReadBytes int64
		stop           chan struct{}
		managerState
	}
	managerState struct {
		cache map[string]*cacheItem
	}
	cacheItem struct {
		ref int32
		ls  *FileLogStream
	}
	// Manager state obj for gob
	managerStateObj struct {
		Cache map[string]*fileLogStreamStateObj
	}
)

func init() {
	gob.Register(&managerStateObj{})
}

func NewManager() *Manager {
	return NewManager2(DefaultLogInputConfig.MaxLineSize, DefaultLogInputConfig.MaxIOReadBytes)
}

func NewManager2(maxLineSize int, maxIOReadBytes int64) *Manager {
	return &Manager{
		managerState: managerState{
			cache: make(map[string]*cacheItem),
		},
		maxLineSize:    maxLineSize,
		maxIOReadBytes: maxIOReadBytes,
		stop:           make(chan struct{}),
	}
}

func (m *Manager) StopAndSaveState(store transfer.StateStore) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	close(m.stop)

	// clean right now
	m.cleanOnce()

	state := &managerStateObj{
		Cache: map[string]*fileLogStreamStateObj{},
	}

	for k, v := range m.cache {
		if s, err := v.ls.SaveState(); err == nil {
			// s == nil means nothing to save
			if s != nil {
				state.Cache[k] = s
				logger.Infoz("[transfer] [logstream] stream save state success", zap.String("path", k))
			}
		} else {
			logger.Errorz("[transfer] [logstream] stream save state error", zap.String("path", k), zap.Error(err))
		}
	}

	store.Put(stateKey, state)

	return nil
}

func (m *Manager) LoadState(store transfer.StateStore) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stateI, err := store.Get(stateKey)
	if err != nil {
		return err
	}

	state := stateI.(*managerStateObj)

	if x := len(m.cache); x != 0 {
		return fmt.Errorf("LogStream.Manager already has cached LogStreams, size=%d", x)
	}

	for key, s := range state.Cache {
		// We first load the LogStream. Now its ref is 1.
		fls := m.acquire0(key).(*FileLogStream)
		if err := fls.LoadState(s); err != nil {
			m.release0(key, fls)
			logger.Errorz("[transfer] [logstream] stream load state error", zap.String("key", key), zap.Error(err))
		} else {
			logger.Infoz("[transfer] [logstream] stream load state success", zap.String("key", key))
		}
	}

	return nil
}

func (m *Manager) Start() {
	go m.cleanLoop()

	http.HandleFunc("/api/lsm/state", func(writer http.ResponseWriter, request *http.Request) {
		json.NewEncoder(writer).Encode(m.State())
	})

	stat.DefaultManager1S.Gauge("lsm.state.totalPendingBytes", func() []stat.GaugeSubItem {
		m.mutex.Lock()
		defer m.mutex.Unlock()

		totalPendingBytes := int64(0)
		for _, item := range m.cache {
			item.ls.mutex.Lock()
			totalPendingBytes += item.ls.pendingBytes
			item.ls.mutex.Unlock()
		}

		return []stat.GaugeSubItem{
			{
				Values: []int64{totalPendingBytes},
			},
		}
	})
}

func (m *Manager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.isStopped() {
		return
	}

	close(m.stop)
}

func (m *Manager) isStopped() bool {
	select {
	case <-m.stop:
		return true
	default:
		return false
	}
}

func (m *Manager) cleanLoop() {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-timer.C:
			func() {
				m.mutex.Lock()
				defer m.mutex.Unlock()
				if m.isStopped() {
					return
				}
				m.cleanOnce()
				timer.Reset(time.Minute)
			}()
		}
	}
}

func (m *Manager) cleanOnce() {
	for _, item := range m.cache {
		item.ls.Clean()
	}
}

func (m *Manager) Acquire(path string) LogStream {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.acquire0(path)
}

func (m *Manager) acquire0(path string) LogStream {
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
	defer m.mutex.Unlock()

	m.release0(path, ls)
}

func (m *Manager) release0(path string, ls LogStream) {
	i := m.cache[path]
	i.ref--
	if i.ref == 0 {
		delete(m.cache, path)
		i.ls.Stop()
	}
}

func (m *Manager) CleanInvalidRefAfterLoadState() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, cache := range m.cache {
		// If this LogStream is refed by any active LogPipeline, it must have ref > 1.
		// Because we call acquire0 in the LoadState func, it will first load LogStream with ref = 1.
		if cache.ref == 1 {
			logger.Infoz("[transfer] remove invalid LogStream with ref = 1", zap.String("key", key))
			delete(m.cache, key)
			cache.ls.Stop()
			continue
		}

		if len(cache.ls.listeners) != cache.ls.matchesSuccessCount {
			// TODO fatal error
			panic("len(cache.ls.listeners) != cache.ls.matchesSuccessCount")
		}
	}
}

func (m *Manager) State() interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	r := map[string]interface{}{}

	totalPendingBytes := int64(0)
	for key, item := range m.cache {
		totalPendingBytes += item.ls.pendingBytes
		if item.ls.file != nil {
			r[key] = map[string]interface{}{
				"ref":          item.ref,
				"pendingReads": item.ls.pendingReads,
				"pendingBytes": item.ls.pendingBytes,
			}
		}
	}

	r["totalPendingBytes"] = totalPendingBytes

	return r
}
