/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"os"
)

const (
	nodeIndexByName = "node:ip"
)

type (
	localAgentMetaImpl struct {
		informer cache.SharedIndexInformer
		store    cache.Indexer
		stopCh   chan struct{}
	}
)

var (
	// Make sure localAgentMetaImpl impl cri.LocalAgentMeta
	_ cri.LocalAgentMeta = &localAgentMetaImpl{}

	nodeIndexers = cache.Indexers{
		nodeIndexByName: func(obj interface{}) ([]string, error) {
			return []string{obj.(*v1.Node).Name}, nil
		},
	}
)

func newLocalAgentMetaImpl(getter cache.Getter) *localAgentMetaImpl {
	// nodeName := os.Getenv("NODE_NAME")
	// selector := fields.OneTermEqualSelector("spec.nodeName", nodeName)
	selector := fields.Everything()
	listWatch := cache.NewListWatchFromClient(getter, "nodes", v1.NamespaceAll, selector)
	informer := cache.NewSharedIndexInformer(listWatch, &v1.Node{}, 0, nodeIndexers)
	return &localAgentMetaImpl{
		informer: informer,
		store:    informer.GetIndexer(),
		stopCh:   make(chan struct{}),
	}
}

func (m *localAgentMetaImpl) start() {
	go m.informer.Run(m.stopCh)
}

func (m *localAgentMetaImpl) stop() {
	close(m.stopCh)
}

func (m *localAgentMetaImpl) Node() *v1.Node {
	objs, err := m.store.ByIndex(nodeIndexByName, m.NodeName())
	if err != nil || len(objs) != 1 {
		return nil
	}
	return objs[0].(*v1.Node)
}

func (m *localAgentMetaImpl) Namespace() string {
	return os.Getenv("POD_NAMESPACE")
}

// PodName returns current agent pod name
func (m *localAgentMetaImpl) PodName() string {
	return os.Getenv("POD_NAME")
}

func (m *localAgentMetaImpl) PodIP() string {
	return os.Getenv("POD_IP")
}

func (m *localAgentMetaImpl) NodeIP() string {
	s := os.Getenv("NODE_IP")
	if s == "" {
		s = os.Getenv("HOST_IP")
	}
	return s
}

func (m *localAgentMetaImpl) NodeName() string {
	return os.Getenv("NODE_NAME")
}

func (m *localAgentMetaImpl) IsLocalPod(pod *v1.Pod) bool {
	return m.NodeName() == pod.Spec.NodeName
}
