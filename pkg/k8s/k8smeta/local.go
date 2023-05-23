/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8smeta

import (
	v1 "k8s.io/api/core/v1"
	"os"
)

type (
	// 当自己也部署在k8s集群中时, 自己在k8s集群里的一些元信息:
	LocalMeta struct{}
)

func (m *LocalMeta) Namespace() string {
	return os.Getenv("POD_NAMESPACE")
}

// PodName returns current agent pod name
func (m *LocalMeta) PodName() string {
	return os.Getenv("POD_NAME")
}

func (m *LocalMeta) PodIP() string {
	return os.Getenv("POD_IP")
}

func (m *LocalMeta) HostIP() string {
	return os.Getenv("HOST_IP")
}

func (m *LocalMeta) NodeIP() string {
	return m.HostIP()
}

func (m *LocalMeta) NodeName() string {
	return os.Getenv("NODE_NAME")
}

// 假设 agent 以 daemonset 模式运行, 判断给定 pod 是否是本物理机上的 pod
func (m *LocalMeta) IsLocalPod(pod *v1.Pod) bool {
	return m.HostIP() == pod.Status.HostIP
}
