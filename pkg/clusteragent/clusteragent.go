/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package clusteragent

import (
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssync"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

type (
	// MasterComponent starts k8ssync.MetaSyncer
	MasterComponent struct {
		k8sMetaSyncer k8ssync.MetaSyncer
	}
)

func (c *MasterComponent) Start() {
	logger.Metaz("[clusteragent] start")

	if c.k8sMetaSyncer != nil {
		return
	}

	k8sMetaSyncer := k8ssync.NewMetaSyncer(ioc.RegistryService, ioc.K8sClientset)
	k8sMetaSyncer.Start()
	c.k8sMetaSyncer = k8sMetaSyncer
}

func (c *MasterComponent) Stop() {
	logger.Metaz("[clusteragent] stop")

	s := c.k8sMetaSyncer
	c.k8sMetaSyncer = nil
	if s != nil {
		s.Stop()
	}
}
