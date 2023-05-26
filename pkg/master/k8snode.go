/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package master

import (
	"fmt"
	"github.com/bep/debounce"
	"github.com/jpillora/backoff"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"sync"
	"time"
)

type (
	// K8sNodeMasterMaintainer selects the agent pod with the smallest nodeName as the master agent pod.
	K8sNodeMasterMaintainer struct {
		k8smm            *k8smeta.K8sLocalMetaManager
		stopCh           chan struct{}
		store            cache.Store
		masterPod        *v1.Pod
		debounce         func(f func())
		masterComponents []Component
		mutex            sync.Mutex
	}
	Component interface {
		Start()
		Stop()
	}
)

func NewK8sNodeMasterMaintainer(k8smm *k8smeta.K8sLocalMetaManager) *K8sNodeMasterMaintainer {
	return &K8sNodeMasterMaintainer{
		k8smm:    k8smm,
		stopCh:   make(chan struct{}, 1),
		debounce: debounce.New(5 * time.Second),
	}
}

// Register components which should be run when current node become mater.
// This func should only be called before calling Start.
func (m *K8sNodeMasterMaintainer) Register(mc Component) {
	m.masterComponents = append(m.masterComponents, mc)
}

func (m *K8sNodeMasterMaintainer) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Listen to holoinsight-agent namespace pods
	agentNamespace := m.k8smm.LocalAgentMeta.Namespace()
	selector := fields.Everything()
	getter := m.k8smm.Clientset.CoreV1().RESTClient()
	listWatch := cache.NewListWatchFromClient(getter, string(v1.ResourcePods), agentNamespace, selector)

	store, controller := cache.NewInformer(listWatch, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { m.onChange() },
		UpdateFunc: func(oldObj, newObj interface{}) { m.onChange() },
		DeleteFunc: func(obj interface{}) { m.onChange() },
	})

	m.store = store

	go controller.Run(m.stopCh)

	b := &backoff.Backoff{
		Factor: 2,
		Jitter: true,
		Min:    50 * time.Millisecond,
		Max:    time.Second,
	}

	for !controller.HasSynced() {
		logger.Infoz("[bootstrap] [master] wait meta sync")
		time.Sleep(b.Duration())
	}

	logger.Infoz("[bootstrap] [master] bootstrap done")
}

func (m *K8sNodeMasterMaintainer) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	masterPod := m.masterPod
	if m.iAmMaster(masterPod) {
		m.onMasterLeave()
	}
	close(m.stopCh)
}

func (m *K8sNodeMasterMaintainer) onChange() {
	logger.Metaz("[master] pod onChange")
	m.debounce(m.onChange0)
}

func (m *K8sNodeMasterMaintainer) onChange0() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isStopped() {
		return
	}

	logger.Metaz("[master] pod onChange0")
	items := m.store.List()

	var selfPod *v1.Pod
	selfPodName := m.k8smm.LocalAgentMeta.PodName()
	for i := range items {
		pod := items[i].(*v1.Pod)
		if pod.Name == selfPodName {
			selfPod = pod
			break
		}
	}

	if selfPod == nil {
		logger.Metaz("[master] no self pod")
		return
	}

	var newMasterPod *v1.Pod
	for i := range items {
		pod := items[i].(*v1.Pod)
		// TODO ugly
		if pod.GenerateName == selfPod.GenerateName {
			// TODO how to skip problem pods?
			if pod.Status.Phase != v1.PodRunning {
				continue
			}
			if newMasterPod == nil {
				newMasterPod = pod
			} else if pod.Spec.NodeName < newMasterPod.Spec.NodeName {
				newMasterPod = pod
			}
		}
	}

	if newMasterPod == nil {
		logger.Metaz("[master] master nil")
		return
	}

	if m.masterPod == nil {
		m.masterPod = newMasterPod
		logger.Metaz("[master] master first",
			zap.String("new", fmt.Sprintf("%s/%s", newMasterPod.Name, string(newMasterPod.UID))))

		if m.iAmMaster(newMasterPod) {
			m.onMasterEnter()
		}

	} else {
		oldMasterPod := m.masterPod
		if oldMasterPod.UID != newMasterPod.UID {
			logger.Metaz("[master] master changed",
				zap.String("old", fmt.Sprintf("%s/%s", oldMasterPod.Name, string(oldMasterPod.UID))),
				zap.String("new", fmt.Sprintf("%s/%s", newMasterPod.Name, string(newMasterPod.UID))))

			m.masterPod = newMasterPod

			if m.iAmMaster(oldMasterPod) {
				m.onMasterLeave()
			}

			if m.iAmMaster(newMasterPod) {
				m.onMasterEnter()
			}
		}
	}

}

func (m *K8sNodeMasterMaintainer) iAmMaster(masterPod *v1.Pod) bool {
	return masterPod != nil && masterPod.Spec.NodeName == m.k8smm.LocalAgentMeta.NodeName()
}

func (m *K8sNodeMasterMaintainer) onMasterEnter() {
	logger.Metaz("[master] master enter")
	for _, component := range m.masterComponents {
		component.Start()
	}
}

func (m *K8sNodeMasterMaintainer) onMasterLeave() {
	logger.Metaz("[master] master leave")
	for _, component := range m.masterComponents {
		component.Stop()
	}
}

func (m *K8sNodeMasterMaintainer) isStopped() bool {
	select {
	case <-m.stopCh:
		return true
	default:
		return false
	}
}
