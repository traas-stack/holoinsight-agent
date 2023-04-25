/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

import (
	"sync"
	"time"
)

type (
	// cache latest timestramp every metric
	timeCache struct {
		cache map[timeCacheKey]time.Time
		mutex sync.RWMutex
	}
	timeCacheKey struct {
		namespace string
		name      string
	}
)

func newTimeCache() *timeCache {
	return &timeCache{
		cache: make(map[timeCacheKey]time.Time),
	}
}

func (c *timeCache) get(namespace, name string) time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.cache[timeCacheKey{namespace: namespace, name: name}]
}

func (c *timeCache) put(namespace, name string, ts time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[timeCacheKey{namespace: namespace, name: name}] = ts
}
