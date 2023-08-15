/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package silence

import (
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type (
	defaultControllerManager struct {
		helper Helper
	}
)

func newDefaultControllerInformer(helper Helper) ControllerManager {
	return &defaultControllerManager{
		helper: helper,
	}
}

func (m *defaultControllerManager) Run(stopCh chan struct{}) {
}

func (m *defaultControllerManager) GetPodStatus(pod *v1.Pod) string {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil {
			return status.State.Waiting.Reason
		}
		if status.State.Terminated != nil {
			return status.State.Terminated.Reason
		}
		if status.State.Running != nil {
			return "Running"
		}
	}
	return string(pod.Status.Phase)
}

func (m *defaultControllerManager) PodHandler() cache.ResourceEventHandler {
	return m
}

func (m *defaultControllerManager) OnAdd(obj interface{}) {
	m.helper.OnPodAdd(obj)
}

func (m *defaultControllerManager) OnUpdate(oldObj, newObj interface{}) {
	old := oldObj.(*v1.Pod)
	pod := newObj.(*v1.Pod)

	status := m.GetPodStatus(pod)
	if "Terminating" == status {
		logger.Metaz("[k8s-op] [pod] terminating", zap.String("namespace", pod.Namespace), zap.String("pod", pod.Name), zap.String("status", status))
		m.helper.EnterSilence(pod)
	} else if !IsReady(old) && IsReady(pod) {
		logger.Metaz("[k8s-op] [pod] ready", zap.String("namespace", pod.Namespace), zap.String("pod", pod.Name), zap.String("status", status))
		m.helper.LeaveSilence(pod)
	} else {
		logger.Metaz("[k8s-op] [pod] update", zap.String("namespace", pod.Namespace), zap.String("pod", pod.Name), zap.String("status", status), zap.String("ready", GetReadyStr(pod)))
	}
}

func (m *defaultControllerManager) OnDelete(obj interface{}) {
	m.helper.OnPodDel(obj)
}

func (m *defaultControllerManager) IsUnderControl(pod *v1.Pod) bool {
	for _, o := range pod.OwnerReferences {
		if o.APIVersion == "apps/v1" && o.Kind == "StatefulSet" {
			return true
		}
		if o.APIVersion == "apps/v1" && o.Kind == "ReplicaSet" {
			return true
		}
	}
	return false
}
