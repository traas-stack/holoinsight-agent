/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type (
	// localPodMeta holds pods of node agent is running on
	localPodMeta interface {
		start()
		stop()
		getAllPods() []*v1.Pod
		addEventHandler(handler cache.ResourceEventHandler)
		getInformer() cache.SharedIndexInformer
	}
)

var (
	podIndexers = cache.Indexers{}
)

func newLocalPodMeta(localNodeName string, getter cache.Getter) localPodMeta {
	if appconfig.StdAgentConfig.K8s.Cri.GetPodsFromKubeletAPI != "" {
		return newLocalPodMetaKubelet(appconfig.StdAgentConfig.K8s.Cri.GetPodsFromKubeletAPI)
	} else {
		return newLocalPodMetaInformer(localNodeName, getter)
	}
}
