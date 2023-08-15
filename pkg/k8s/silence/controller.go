/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package silence

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type (
	ControllerManager interface {
		PodHandler() cache.ResourceEventHandler
		Run(stopCh chan struct{})
		IsUnderControl(pod *v1.Pod) bool
	}
)
