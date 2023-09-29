/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sync"
	"time"
)

type (
	defaultMetaStore struct {
		state          *internalState
		localPodMeta   localPodMeta
		localAgentMeta *localAgentMetaImpl
		listeners      []cri.MetaListener
		mutex          sync.Mutex
	}

	internalState struct {
		pods                 []*cri.Pod
		runningPodMap        map[string]*cri.Pod
		containerMap         map[string]*cachedContainer
		shortCidContainerMap map[string]*cachedContainer
		// podByKey pod map by key("${ns}/${pod}")
		podByKey       map[string]*cri.Pod
		podByUID       map[types.UID]*cri.Pod
		podBySandboxId map[string]*cri.Pod
	}

	cachedContainer struct {
		engineContainer *cri.EngineDetailContainer
		criContainer    *cri.Container
	}
)

var (
	// Make sure defaultMetaStore impl cri.MetaStore
	_ cri.MetaStore = &defaultMetaStore{}
)

func newDefaultMetaStore(clientset *kubernetes.Clientset) *defaultMetaStore {
	getter := clientset.CoreV1().RESTClient()
	lm := newLocalAgentMetaImpl(getter)
	return &defaultMetaStore{
		localAgentMeta: lm,
		state:          newInternalState(),
		localPodMeta:   newLocalPodMeta(lm.NodeName(), getter),
	}
}

func (e *defaultMetaStore) AddListener(listener cri.MetaListener) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.listeners = append(e.listeners, listener)
}

func (e *defaultMetaStore) RemoveListener(listener cri.MetaListener) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	newOne := make([]cri.MetaListener, 0, len(e.listeners))
	for _, l := range e.listeners {
		if l != listener {
			newOne = append(newOne, l)
		}
	}
	e.listeners = newOne
}

func (e *defaultMetaStore) LocalAgentMeta() cri.LocalAgentMeta {
	return e.localAgentMeta
}

func (e *defaultMetaStore) GetAllPods() []*cri.Pod {
	return e.state.pods
}

func (e *defaultMetaStore) GetPodBySandboxId(sandboxId string) (*cri.Pod, error) {
	pod, ok := e.state.podBySandboxId[sandboxId]
	if ok {
		return pod, nil
	}
	return nil, fmt.Errorf("no pod sandboxId=[%s]", sandboxId)
}

func (e *defaultMetaStore) GetContainerByCid(cid string) (*cri.Container, bool) {
	state := e.state
	// docker short container id length = 12
	// fa5799111150
	if c, ok := state.shortCidContainerMap[cid]; ok {
		return c.criContainer, true
	}
	// docker full container id
	if c, ok := state.containerMap[cid]; ok {
		return c.criContainer, true
	}
	return nil, false
}

func (e *defaultMetaStore) GetPod(ns, pod string) (*cri.Pod, error) {
	state := e.state
	p, ok := state.podByKey[ns+"/"+pod]
	if !ok {
		return nil, cri.NoPodError(ns, pod)
	}
	return p, nil
}

func (e *defaultMetaStore) Start() error {
	e.localAgentMeta.start()
	e.localPodMeta.start()

	b := &backoff.Backoff{
		Factor: 2,
		Jitter: true,
		Min:    50 * time.Millisecond,
		Max:    time.Second,
	}

	controllers := []cache.Controller{e.localAgentMeta.informer}
	if i := e.localPodMeta.getInformer(); i != nil {
		controllers = append(controllers, i)
	}
	for _, controller := range controllers {
		for !controller.HasSynced() {
			logger.Infoz("[bootstrap] [k8s] [meta] wait meta sync")
			time.Sleep(b.Duration())
		}
	}

	return nil
}

func (e *defaultMetaStore) Stop() {
	e.localPodMeta.stop()
	e.localAgentMeta.stop()
}

func (e *defaultMetaStore) getCriPods(pods []*v1.Pod) []*cri.Pod {
	state := e.state
	criPods := make([]*cri.Pod, 0, len(pods))
	for _, pod := range pods {
		if criPod, ok := state.podByUID[pod.UID]; ok {
			criPods = append(criPods, criPod)
		}
	}
	return criPods
}

func newInternalState() *internalState {
	return &internalState{
		runningPodMap:        make(map[string]*cri.Pod),
		containerMap:         make(map[string]*cachedContainer),
		shortCidContainerMap: make(map[string]*cachedContainer),
		podByKey:             make(map[string]*cri.Pod),
		podByUID:             make(map[types.UID]*cri.Pod),
		podBySandboxId:       make(map[string]*cri.Pod),
	}
}

func (s *internalState) build() {
	for _, c := range s.containerMap {
		s.shortCidContainerMap[c.criContainer.ShortContainerID()] = c
	}
	for _, pod := range s.pods {
		if pod.IsRunning() {
			s.runningPodMap[pod.Namespace+"/"+pod.Name] = pod
		}
		s.podByKey[pod.Namespace+"/"+pod.Name] = pod
		s.podByUID[pod.UID] = pod
		for _, container := range pod.All {
			cri.SortMountPointsByLongSourceFirst(container.Mounts)
		}
		if pod.Sandbox != nil {
			s.podBySandboxId[pod.Sandbox.Id] = pod
			for _, container := range pod.All {
				if pod.Sandbox != container {
					container.NetworkMode = pod.Sandbox.NetworkMode
				}
			}
		}
	}
}
