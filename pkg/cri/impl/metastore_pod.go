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
	// localPodMeta holds pods of node agent is running on
	localPodMeta struct {
		informer cache.SharedIndexInformer
		store    cache.Indexer
		stopCh   chan struct{}
	}
)

var (
	podIndexers = cache.Indexers{}
)

func newPodMeta(localNodeName string, getter cache.Getter) *localPodMeta {
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
	return &localPodMeta{
		informer: informer,
		store:    informer.GetIndexer(),
		stopCh:   make(chan struct{}),
	}
}

func (m *localPodMeta) start() {
	go m.informer.Run(m.stopCh)
}

func (m *localPodMeta) stop() {
	close(m.stopCh)
}

func (m *localPodMeta) getAllPods() []*v1.Pod {
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

func (m *localPodMeta) addEventHandler(handler cache.ResourceEventHandler) {
	m.informer.AddEventHandler(handler)
}
