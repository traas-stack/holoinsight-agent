/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8ssysmetrics

import (
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics/common"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/transfer"
)

type (
	// sysCollectorPipeline is the api.Pipeline adapter for common.SysCollector
	sysCollectorPipeline struct {
		key       string
		collector common.SysCollector
	}
)

func NewSysCollectorPipeline(key string, collector common.SysCollector) api.Pipeline {
	return &sysCollectorPipeline{
		key:       key,
		collector: collector,
	}
}

func (s *sysCollectorPipeline) Key() string {
	return s.key
}

func (s *sysCollectorPipeline) Start() error {
	s.collector.Start()
	return nil
}

func (s *sysCollectorPipeline) Stop() {
	s.collector.Stop()
}

func (s *sysCollectorPipeline) SetupConsumer(st *api.SubTask) error {
	return nil
}

func (s *sysCollectorPipeline) Update(f func(api.Pipeline)) {
	f(s)
}

func (s *sysCollectorPipeline) View(f func(api.Pipeline)) {
	f(s)
}

func (p *sysCollectorPipeline) StopAndSaveState(store transfer.StateStore) error {
	p.collector.Stop()

	if si, ok := p.collector.(transfer.StatefulInput); ok {
		if state, err := si.SaveState(); err != nil {
			return err
		} else {
			store.Put("pipeline@"+p.key+"@collector", state)
		}
	}

	return nil
}

func (p *sysCollectorPipeline) LoadState(store transfer.StateStore) error {
	state, err := store.Get("pipeline@" + p.key + "@collector")
	if err != nil {
		return err
	}

	if si, ok := p.collector.(transfer.StatefulInput); ok {
		if err := si.LoadState(state); err != nil {
			return err
		}
	}

	return nil
}
