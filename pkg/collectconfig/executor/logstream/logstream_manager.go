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
	// Manager 用于确保相同路径的path只有一个 LogStream 实例
	Manager struct {
		mutex sync.Mutex
		stop  chan struct{}
		managerState
	}
	managerState struct {
		cache map[string]*managerCachedItem
	}
	managerCachedItem struct {
		ref int32
		ls  *GLogStream
	}
	// Manager state obj for gob
	managerStateObj struct {
		Cache []*cachedLogStreamStateObj
	}
	cachedLogStreamStateObj struct {
		Key            string
		SlsConfig      *SlsConfig
		LogStreamState interface{}
	}
)

func init() {
	gob.Register(&managerStateObj{})
	//gob.Register(&cachedLogStreamStateObj{})
	gob.Register(&fileStateObj{})
}

func NewManager() *Manager {
	return &Manager{
		managerState: managerState{
			cache: make(map[string]*managerCachedItem),
		},
		stop: make(chan struct{}),
	}
}

func (m *Manager) StopAndSaveState(store transfer.StateStore) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	close(m.stop)

	// clean right now
	m.cleanOnce()

	state := &managerStateObj{}

	for k, v := range m.cache {
		if s, err := v.ls.SaveState(); err == nil {
			// s == nil means nothing to save
			if s != nil {
				var sc *SlsConfig
				if x, ok := v.ls.sub.(*slsSubLogStream); ok {
					sc = &x.config
				}
				state.Cache = append(state.Cache, &cachedLogStreamStateObj{
					Key:            k,
					SlsConfig:      sc,
					LogStreamState: s,
				})
				logger.Infoz("[transfer] [logstream] stream save state success", zap.String("key", k))
			}
		} else {
			logger.Errorz("[transfer] [logstream] stream save state error", zap.String("key", k), zap.Error(err))
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

	for _, ms := range state.Cache {
		key := ms.Key
		// We first load the LogStream. Now its ref is 1.
		ls := m.acquire0(key, ms.SlsConfig)
		if x, ok := ls.(LogStreamState); ok {
			if err := x.LoadState(ms.LogStreamState); err != nil {
				m.release0(key, ls)
				logger.Errorz("[transfer] [logstream] stream load state error", zap.String("key", key), zap.Error(err))
			} else {
				logger.Infoz("[transfer] [logstream] stream load state success", zap.String("key", key))
			}
		} else {
			logger.Infoz("[transfer] [logstream] skip load state", zap.String("key", key))
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
			totalPendingBytes += item.ls.Stat().PendingBytes
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

func (m *Manager) AcquireFile(path string) LogStream {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.acquire0(path, nil)
}

func (m *Manager) AcquireSls(config SlsConfig) LogStream {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.acquire0(config.BuildKey(), &config)
}

func (m *Manager) acquire0(key string, sc *SlsConfig) LogStream {
	i := m.cache[key]
	if i != nil {
		i.ref++
		return i.ls
	}

	var ls *GLogStream
	if sc != nil {
		ls = NewSlsLogStream(*sc)
	} else {
		ls = NewFileLogStream(FileConfig{
			Path:           key,
			MaxLineSize:    DefaultFileConfig.MaxLineSize,
			MaxIOReadBytes: DefaultFileConfig.MaxIOReadBytes,
		})
	}
	ls.Start()
	i = &managerCachedItem{
		ref: 1,
		ls:  ls,
	}
	m.cache[key] = i
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
	}
}

func (m *Manager) State() interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	r := map[string]interface{}{}

	totalPendingBytes := int64(0)
	for key, item := range m.cache {
		st := item.ls.Stat()
		totalPendingBytes += st.PendingBytes
		r[key] = map[string]interface{}{
			"ref":          item.ref,
			"pendingReads": st.PendingReads,
			"pendingBytes": st.PendingBytes,
		}
	}

	r["totalPendingBytes"] = totalPendingBytes

	return r
}
