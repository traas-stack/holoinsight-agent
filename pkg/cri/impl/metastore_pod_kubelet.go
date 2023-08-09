/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"encoding/json"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultPullInterval = 10 * time.Second
)

type (
	localPodMetaKubelet struct {
		kubeletPodsUrl string
		stopCh         chan struct{}
		state          *kubeletInternalState
		mutex          sync.Mutex
		handlers       []cache.ResourceEventHandler
	}
	kubeletInternalState struct {
		pods   []*v1.Pod
		podMap map[types.UID]*v1.Pod
	}
)

func (l *localPodMetaKubelet) getInformer() cache.SharedIndexInformer {
	return nil
}

func (l *localPodMetaKubelet) start() {
	for {
		err := l.updateOnce()
		if err == nil {
			break
		}
		logger.Errorz("[kubelet] init load pods error", zap.Error(err))
		time.Sleep(time.Second)
	}

	go l.loop()
}

func (l *localPodMetaKubelet) loop() {
	timer := time.NewTimer(defaultPullInterval)
	defer timer.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case <-timer.C:
			func() {
				l.mutex.Lock()
				defer l.mutex.Unlock()
				if l.isStopped() {
					return
				}

				l.updateOnce()
				timer.Reset(defaultPullInterval)
			}()
		}
	}
}

func (l *localPodMetaKubelet) isStopped() bool {
	select {
	case <-l.stopCh:
		return true
	default:
		return false
	}
}

func (l *localPodMetaKubelet) stop() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.isStopped() {
		return
	}

	close(l.stopCh)
}

func (l *localPodMetaKubelet) getAllPods() []*v1.Pod {
	state := l.state
	return state.pods
}

func (l *localPodMetaKubelet) getPodsFromKubelet() ([]*v1.Pod, error) {
	resp, err := http.Get(l.kubeletPodsUrl)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	podList := &v1.PodList{}
	if err = json.NewDecoder(resp.Body).Decode(podList); err != nil {
		return nil, err
	}

	pods := make([]*v1.Pod, len(podList.Items))
	for i, pod0 := range podList.Items {
		pod := pod0
		pods[i] = &pod
	}
	return pods, nil
}

func (l *localPodMetaKubelet) isUpdated(o, n *v1.Pod) bool {
	if o.ResourceVersion != n.ResourceVersion {
		return true
	}
	if o.Status.Phase != n.Status.Phase {
		return true
	}
	if len(o.Status.ContainerStatuses) != len(n.Status.ContainerStatuses) {
		return true
	}
	for i, os := range o.Status.ContainerStatuses {
		ns := n.Status.ContainerStatuses[i]
		if os.State.Running != nil && (ns.State.Running == nil || os.State.Running.StartedAt != ns.State.Running.StartedAt) {
			return true
		}
		if os.State.Terminated != nil && (ns.State.Terminated == nil || os.State.Terminated.StartedAt != ns.State.Terminated.StartedAt) {
			return true
		}
	}
	return false
}

func (l *localPodMetaKubelet) updateOnce() error {
	oldState := l.state

	oldPods := make(map[types.UID]*v1.Pod, len(oldState.podMap))
	for k, v := range oldState.podMap {
		oldPods[k] = v
	}

	newPods, err := l.getPodsFromKubelet()
	if err != nil {
		logger.Metaz("[kubelet] sync once", zap.Error(err))
		return err
	}

	newPodMap := make(map[types.UID]*v1.Pod)

	var add, update []*v1.Pod
	for _, pod := range newPods {
		newPodMap[pod.UID] = pod

		oldPod := oldPods[pod.UID]
		delete(oldPods, pod.UID)

		switch {
		case oldPod == nil:
			add = append(add, pod)
		case l.isUpdated(oldPod, pod):
			// nothing changed
			update = append(update, pod)
		}
		if oldPod != nil {
			if pod.Name == "cadvisor-t5hnx" {
				logger.Metaz("aaaa", zap.String("oldV", oldPod.ResourceVersion), zap.String("newV", pod.ResourceVersion))
			}
		}
	}

	l.state = &kubeletInternalState{
		pods:   newPods,
		podMap: newPodMap,
	}

	for _, pod := range add {
		for _, handler := range l.handlers {
			handler.OnAdd(pod)
		}
	}
	for _, pod := range update {
		for _, handler := range l.handlers {
			handler.OnUpdate(oldState.podMap[pod.UID], pod)
		}
	}
	for _, pod := range oldPods {
		for _, handler := range l.handlers {
			handler.OnDelete(pod)
		}
	}

	logger.Metaz("[kubelet] sync once", zap.Int("pods", len(newPods)), zap.Int("add", len(add)), zap.Int("update", len(update)), zap.Int("del", len(oldPods)))

	return nil
}

func (l *localPodMetaKubelet) addEventHandler(handler cache.ResourceEventHandler) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.handlers = append(l.handlers, handler)
}

func newLocalPodMetaKubelet(kubeletPodsUrl string) *localPodMetaKubelet {
	if strings.Contains(kubeletPodsUrl, "%s") {
		nodeIp := os.Getenv("NODE_IP")
		kubeletPodsUrl = fmt.Sprintf(kubeletPodsUrl, nodeIp)
	}
	return &localPodMetaKubelet{
		kubeletPodsUrl: kubeletPodsUrl,
		stopCh:         make(chan struct{}),
		state: &kubeletInternalState{
			pods:   nil,
			podMap: make(map[types.UID]*v1.Pod),
		},
	}
}
