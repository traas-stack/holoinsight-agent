/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

type (
	localPodMetaInformer struct {
		informer cache.SharedIndexInformer
		store    cache.Indexer
		stopCh   chan struct{}
	}
)

func (m *localPodMetaInformer) start() {
	go m.informer.Run(m.stopCh)
}

func (m *localPodMetaInformer) stop() {
	close(m.stopCh)
}

func (m *localPodMetaInformer) getAllPods() []*v1.Pod {
	objs := m.store.List()
	return toPods(objs, nil)
}

func toPods(objs []interface{}, err error) []*v1.Pod {
	if err != nil {
		return nil
	}
	pods := make([]*v1.Pod, len(objs))
	for i := range objs {
		pods[i] = objs[i].(*v1.Pod)
	}
	return pods
}

func (m *localPodMetaInformer) addEventHandler(handler cache.ResourceEventHandler) {
	m.informer.AddEventHandler(handler)
}

func (m *localPodMetaInformer) getInformer() cache.SharedIndexInformer {
	return m.informer
}

func newLocalPodMetaInformer(localNodeName string, getter cache.Getter) *localPodMetaInformer {
	if localNodeName == "" {
		panic("local nodeName is empty")
	}
	// Only subscribe pods which belongs to current node
	selector := fields.OneTermEqualSelector("spec.nodeName", localNodeName)
	listWatch := cache.NewListWatchFromClient(getter, string(v1.ResourcePods), v1.NamespaceAll, selector)
	informer := cache.NewSharedIndexInformer(listWatch, &v1.Pod{}, 0, podIndexers)
	informer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
		logger.Errorf("[meta] watch error %+v", err)
	})
	return &localPodMetaInformer{
		informer: informer,
		store:    informer.GetIndexer(),
		stopCh:   make(chan struct{}),
	}
}
