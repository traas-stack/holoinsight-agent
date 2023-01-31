package providers

import (
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collecttask"
	"sync"
)

type (
	InputProvider func(task *collecttask.CollectTask) (interface{}, error)
)

var (
	providers = make(map[string]InputProvider)
	mutex     sync.RWMutex
)

func Register(configType string, p InputProvider) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := providers[configType]; ok {
		panic(fmt.Errorf("duplicated input provider: %s", configType))
	}

	providers[configType] = p
}

func Get(configType string) (InputProvider, bool) {
	mutex.RLock()
	defer mutex.RUnlock()

	p, ok := providers[configType]
	return p, ok
}
