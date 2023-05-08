/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package inputproxy

import (
	api2 "github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"sync"
)

type (
	Provider func() api2.InputExtNsEnter
)

var (
	providers = make(map[string]Provider)
	mutex     sync.RWMutex
)

// Register associates name to provider.
func Register(name string, provider Provider) {
	mutex.Lock()
	defer mutex.Unlock()
	providers[name] = provider
}

// Get returns provider associated with the name
func Get(name string) Provider {
	mutex.RLock()
	defer mutex.RUnlock()

	return providers[name]
}
