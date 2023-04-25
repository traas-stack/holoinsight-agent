/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	cms20190101 "github.com/alibabacloud-go/cms-20190101/v7/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/alibabacloud-go/tea/tea"
	"sync"
	"sync/atomic"
	"time"
)

type (
	clientManager struct {
		mutex sync.Mutex
		cache sync.Map
	}
	cacheItem struct {
		client         *cms20190101.Client
		err            error
		lastAccessTime int64
	}
)

func (m *clientManager) start() {
	go func() {
		for range time.NewTicker(time.Minute).C {
			m.clearExpired()
		}
	}()
}

func (m *clientManager) getClient(endpoint, ak, sk string) (*cms20190101.Client, error) {
	ci := m.getClient0(endpoint, ak, sk)
	atomic.StoreInt64(&ci.lastAccessTime, util.CurrentMS())
	return ci.client, ci.err
}

func (m *clientManager) getClient0(endpoint, ak, sk string) *cacheItem {
	cacheKey := fmt.Sprintf("%s_%s_%s", endpoint, ak, sk)

	if i, ok := m.cache.Load(cacheKey); ok {
		return i.(*cacheItem)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// double check
	if i, ok := m.cache.Load(cacheKey); ok {
		return i.(*cacheItem)
	}

	// create
	config := &openapi.Config{
		AccessKeyId:     tea.String(ak),
		AccessKeySecret: tea.String(sk),
		Endpoint:        tea.String(endpoint),
	}
	client, err := cms20190101.NewClient(config)
	ci := &cacheItem{
		client:         client,
		err:            err,
		lastAccessTime: 0,
	}
	m.cache.Store(cacheKey, ci)
	return ci
}

func (m *clientManager) clearExpired() {
	expiredTime := util.CurrentMS() - (10 * time.Minute).Milliseconds()
	m.cache.Range(func(key, value interface{}) bool {
		ci := value.(*cacheItem)
		if ci.lastAccessTime < expiredTime {
			m.cache.Delete(key)
		}
		return true
	})
}
