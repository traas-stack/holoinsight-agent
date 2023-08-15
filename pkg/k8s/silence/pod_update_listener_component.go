/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package silence

import (
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
)

type (
	PodUpdateListenerMasterComponent struct {
		podDeployListener *PodUpdateListener
	}
)

func (c *PodUpdateListenerMasterComponent) Start() {
	c.podDeployListener = NewPodOperationListener(ioc.K8sClientset)
	c.podDeployListener.Start()
}
func (c *PodUpdateListenerMasterComponent) Stop() {
	x := c.podDeployListener
	c.podDeployListener = nil
	if x != nil {
		x.Stop()
	}
}
