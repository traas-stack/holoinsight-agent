/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8slabels

const (
	LabelNamespace     = "io.kubernetes.pod.namespace"
	LabelContainerName = "io.kubernetes.container.name"
	LabelPodName       = "io.kubernetes.pod.name"
	LabelK8sSandboxID  = "io.kubernetes.sandbox.id"
	LabelK8sApp        = "app.kubernetes.io/name"
	// 物理机上特有的几个标记
	LabelK8sRegion           = "topology.kubernetes.io/region"
	LabelK8sZone             = "topology.kubernetes.io/zone"
	LabelK8sOs               = "kubernetes.io/os"
	LabelK8sArch             = "kubernetes.io/arch"
	LabelK8sHostname         = "kubernetes.io/hostname"
	LabelK8sNodeInstanceType = "node.kubernetes.io/instance-type"
	LabelApp                 = "app"
)

// 我们对app的获取方式有特殊定义, 不允许直接调用该方法取获取
func GetApp(labels map[string]string) string {
	if a, ok := labels[LabelK8sApp]; ok && a != "" {
		return a
	}
	return labels[LabelApp]
}

func GetNamespace(labels map[string]string) string {
	return labels[LabelNamespace]
}

func GetPodName(labels map[string]string) string {
	return labels[LabelPodName]
}

func GetContainerName(labels map[string]string) string {
	return labels[LabelContainerName]
}

func GetRegion(labels map[string]string) string {
	return labels[LabelK8sRegion]
}

func GetZone(labels map[string]string) string {
	return labels[LabelK8sZone]
}

func GetSandboxID(labels map[string]string) string {
	return labels[LabelK8sSandboxID]
}
