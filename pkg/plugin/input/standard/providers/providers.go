/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package providers

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"strings"
	"sync"
)

type (
	PipelineProvider func(task *collecttask.CollectTask) (api.Pipeline, error)
	InputProvider    func(task *collecttask.CollectTask) (api.Input, error)
)

var (
	inputProviders    = make(map[string]InputProvider)
	pipelineProviders = make(map[string]PipelineProvider)
	mutex             sync.RWMutex
)

func StandardizeType(t string) string {
	index := strings.LastIndexByte(t, '.')
	if index >= 0 {
		t = t[index+1:]
	}
	return strings.ToLower(t)
}

func RegisterInputProvider(configType string, p InputProvider) {
	configType = StandardizeType(configType)

	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := inputProviders[configType]; ok {
		panic(fmt.Errorf("duplicated input provider: %s", configType))
	}

	inputProviders[configType] = p
}

func GetInputProvider(configType string) (InputProvider, bool) {
	configType = StandardizeType(configType)

	mutex.RLock()
	defer mutex.RUnlock()

	p, ok := inputProviders[configType]
	return p, ok
}

func RegisterPipelineFactory(configType string, provider PipelineProvider) {
	configType = StandardizeType(configType)
	mutex.Lock()
	defer mutex.Unlock()

	pipelineProviders[configType] = provider
}

func GetPipelineProvider(configType string) (PipelineProvider, bool) {
	configType = StandardizeType(configType)

	mutex.RLock()
	defer mutex.RUnlock()

	p, ok := pipelineProviders[configType]
	return p, ok
}
