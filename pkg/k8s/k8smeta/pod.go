/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8smeta

import (
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// 同步k8s的元信息
const (
	PodIndexByNamespace    = "pod:namespace"
	PodIndexByNamespacePod = "pod:namespace/pod"
	PodIndexByHostIP       = "pod:hostIP"
	PodIndexByIP           = "pod:ip"
)

type (
	// LocalPodMeta holds pods of node agent is running on
	LocalPodMeta struct {
		informer cache.SharedIndexInformer
		store    cache.Indexer
		stopCh   chan struct{}
	}
)

var (
	// 这里无论如何不能error, 否则panic
	podIndexers = cache.Indexers{
		PodIndexByNamespace: func(obj interface{}) ([]string, error) {
			return []string{obj.(*v1.Pod).Namespace}, nil
		},
		PodIndexByNamespacePod: func(obj interface{}) ([]string, error) {
			pod := obj.(*v1.Pod)
			return []string{pod.Namespace + "/" + pod.Name}, nil
		},
		PodIndexByHostIP: func(obj interface{}) ([]string, error) {
			pod := obj.(*v1.Pod)
			return []string{pod.Status.HostIP}, nil
		},
		PodIndexByIP: func(obj interface{}) ([]string, error) {
			pod := obj.(*v1.Pod)
			return []string{pod.Status.PodIP}, nil
		},
	}
)

func newPodMeta(localNodeName string, getter cache.Getter) *LocalPodMeta {
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
	return &LocalPodMeta{
		informer: informer,
		store:    informer.GetIndexer(),
		stopCh:   make(chan struct{}),
	}
}

func (m *LocalPodMeta) start() {
	go m.informer.Run(m.stopCh)
}

func (m *LocalPodMeta) stop() {
	close(m.stopCh)
}

func (m *LocalPodMeta) GetPodsByNamespace(ns string) []*v1.Pod {
	objs, err := m.store.ByIndex(PodIndexByNamespace, ns)
	return toPods(objs, err)
}

func (m *LocalPodMeta) GetPodsByHostIP(hostIP string) []*v1.Pod {
	objs, err := m.store.ByIndex(PodIndexByHostIP, hostIP)
	return toPods(objs, err)
}

func (m *LocalPodMeta) GetPodsByIP(ip string) []*v1.Pod {
	objs, err := m.store.ByIndex(PodIndexByIP, ip)
	return toPods(objs, err)
}

func (m *LocalPodMeta) GetPodsByApp(namespace, app string) []*v1.Pod {
	objs, err := m.store.ByIndex(PodIndexByNamespace, namespace)
	pods := toPods(objs, err)

	var filtered []*v1.Pod
	for _, pod := range pods {
		if k8smetaextractor.DefaultPodMetaService.ExtractApp(pod) == app {
			filtered = append(filtered, pod)
		}
	}
	return filtered
}

func (m *LocalPodMeta) GetPodByName(ns, pod string) *v1.Pod {
	objs, err := m.store.Index(PodIndexByNamespacePod, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      pod,
		},
	})
	if err != nil || len(objs) == 0 {
		return nil
	}
	return objs[0].(*v1.Pod)
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

func (m *LocalPodMeta) AddEventHandler(handler cache.ResourceEventHandler) {
	m.informer.AddEventHandler(handler)
}
