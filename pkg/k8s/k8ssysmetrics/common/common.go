package common

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	k8smetaextractor "github.com/TRaaSStack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	v1 "k8s.io/api/core/v1"
)

func ExtractPodCommonTags(pod *v1.Pod) map[string]string {
	return map[string]string{
		"namespace": pod.Namespace,
		"pod":       pod.Name,
		"ip":        pod.Status.PodIP,
		"hostname":  k8smetaextractor.DefaultPodMetaService.ExtractHostname(pod),
		"app":       k8smetaextractor.DefaultPodMetaService.ExtractApp(pod),
	}
}

func ExtractContainerCommonTags(container *cri.Container) map[string]string {
	tags := ExtractPodCommonTags(container.Pod.Pod)
	tags["container"] = container.Name
	return tags
}
