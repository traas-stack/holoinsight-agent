package k8smeta

import (
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8sutils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

const (
	NodeIndexByIp = "node:ip"
)

type (
	NodeMeta struct {
		informer cache.SharedIndexInformer
		store    cache.Indexer
		stopCh   chan struct{}
	}
)

var (
	nodeIndexers = cache.Indexers{
		NodeIndexByIp: func(obj interface{}) ([]string, error) {
			node := obj.(*v1.Node)
			ip := k8sutils.GetNodeIP(node)
			// If we return err here, it will lead to panic in k8s internal !!!
			return []string{ip}, nil
		},
	}
)

func newNodeMeta(getter cache.Getter) *NodeMeta {
	listWatch := cache.NewListWatchFromClient(getter, "nodes", v1.NamespaceAll, fields.Everything())
	informer := cache.NewSharedIndexInformer(listWatch, &v1.Node{}, 0, nodeIndexers)
	return &NodeMeta{
		informer: informer,
		store:    informer.GetIndexer(),
		stopCh:   make(chan struct{}),
	}
}

func (m *NodeMeta) start() {
	go m.informer.Run(m.stopCh)
}

func (m *NodeMeta) stop() {
	close(m.stopCh)
}

func (m *NodeMeta) GetNodeByIp(ip string) *v1.Node {
	objs, err := m.store.ByIndex(NodeIndexByIp, ip)
	if err != nil || len(objs) == 0 {
		return nil
	}
	return objs[0].(*v1.Node)
}

func (m *NodeMeta) GetAllNodes() []*v1.Node {
	list := m.store.List()
	nodes := make([]*v1.Node, len(list))
	for i := range list {
		nodes[i] = list[i].(*v1.Node)
	}
	return nodes
}
