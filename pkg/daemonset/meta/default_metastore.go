/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package meta

import (
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
)

type (
	defaultMetaStore struct {
		state *internalState
	}

	internalState struct {
		Pods                 []*cri.Pod
		RunningPodMap        map[string]*cri.Pod         `json:"-"`
		ContainerMap         map[string]*cachedContainer `json:"-"`
		shortCidContainerMap map[string]*cachedContainer
		// podByKey pod map by key("${ns}/${pod}")
		podByKey map[string]*cri.Pod
		// podByHostname pod map by hostname
		podByHostname map[string]*cri.Pod
	}

	cachedContainer struct {
		engineContainer *cri.EngineDetailContainer
		criContainer    *cri.Container
	}
)

func (e *defaultMetaStore) GetAllPods() []*cri.Pod {
	return e.state.Pods
}

func (e *defaultMetaStore) GetContainerByCid(cid string) (*cri.Container, bool) {
	// docker short container id length = 12
	// fa5799111150
	if c, ok := e.state.shortCidContainerMap[cid]; ok {
		return c.criContainer, true
	}
	// docker full container id
	if c, ok := e.state.ContainerMap[cid]; ok {
		return c.criContainer, true
	}
	return nil, false
}

func (e *defaultMetaStore) GetPod(ns, pod string) (*cri.Pod, bool) {
	state := e.state
	p, ok := state.podByKey[ns+"/"+pod]
	return p, ok
}

func (e *defaultMetaStore) GetPodE(ns, pod string) (*cri.Pod, error) {
	p, ok := e.GetPod(ns, pod)
	if !ok {
		return nil, cri.NoPodError(ns, pod)
	}
	return p, nil
}

func (e *defaultMetaStore) GetPodByHostname(hostname string) (*cri.Pod, bool) {
	state := e.state
	p, ok := state.podByHostname[hostname]
	return p, ok
}

func (e *defaultMetaStore) Start() error {
	return nil
}

func (e *defaultMetaStore) Stop() {

}

func newInternalState() *internalState {
	return &internalState{
		RunningPodMap:        make(map[string]*cri.Pod),
		ContainerMap:         make(map[string]*cachedContainer),
		shortCidContainerMap: make(map[string]*cachedContainer),
		podByKey:             make(map[string]*cri.Pod),
		podByHostname:        make(map[string]*cri.Pod),
	}
}

func (s *internalState) build() {
	for _, c := range s.ContainerMap {
		s.shortCidContainerMap[c.criContainer.ShortContainerID()] = c
	}
	s.RunningPodMap = make(map[string]*cri.Pod)
	for _, pod := range s.Pods {
		if pod.IsRunning() {
			s.RunningPodMap[pod.Namespace+"/"+pod.Name] = pod
		}
		s.podByKey[pod.Namespace+"/"+pod.Name] = pod
		hostname := k8smetaextractor.DefaultPodMetaService.ExtractHostname(pod.Pod)
		if hostname != "" {
			// the hostname may be duplicated
			s.podByHostname[hostname] = pod
		}

		for _, container := range pod.All {
			cri.SortMountPointsByLongSourceFirst(container.Mounts)
		}
		if pod.Sandbox != nil {
			for _, container := range pod.All {
				if pod.Sandbox != container {
					container.NetworkMode = pod.Sandbox.NetworkMode
				}
			}
		}
	}
}
