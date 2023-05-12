/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package meta

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8slabels"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	v1 "k8s.io/api/core/v1"
)

func AttachSystemCommonTagsTo(to map[string]string) {
	if to == nil {
		return
	}
	if appconfig.StdAgentConfig.Mode == core.AgentModeDaemonset {
		if !util.StringSliceContains(appconfig.StdAgentConfig.Data.Metric.SuppressedTags, "workspace") {
			to["workspace"] = appconfig.StdAgentConfig.Workspace
		}
	} else {
		to["workspace"] = appconfig.StdAgentConfig.Workspace
	}
}

func ExtractPodCommonTags(pod *v1.Pod) map[string]string {
	tags := map[string]string{
		"namespace": pod.Namespace,
		"pod":       pod.Name,
		"ip":        pod.Status.PodIP,
		"hostname":  k8smetaextractor.DefaultPodMetaService.ExtractHostname(pod),
		"app":       k8smetaextractor.DefaultPodMetaService.ExtractApp(pod),
	}
	RefLabels(appconfig.StdAgentConfig.Data.Metric.RefLabels.Items, pod.Labels, tags)
	AttachSystemCommonTagsTo(tags)

	SuppressCommonTags(tags)
	return tags
}

func ExtractContainerCommonTags(container *cri.Container) map[string]string {
	tags := ExtractPodCommonTags(container.Pod.Pod)
	tags["container"] = container.Name
	return tags
}

func ExtractSidecarTags() map[string]string {
	tags := make(map[string]string, 3)
	tags["app"] = appconfig.StdAgentConfig.App
	tags["ip"] = util.GetLocalIp()
	tags["hostname"] = util.GetHostname()
	AttachSystemCommonTagsTo(tags)
	return tags
}

func ExtractNodeCommonTags(node *v1.Node) map[string]string {
	return map[string]string{
		"name":         node.Name,
		"hostname":     k8smetaextractor.PodMetaServiceInstance.NodeHostname(node),
		"region":       k8slabels.GetRegion(node.Labels),
		"zone":         k8slabels.GetZone(node.Labels),
		"os":           node.Labels[k8slabels.LabelK8sOs],
		"arch":         node.Labels[k8slabels.LabelK8sArch],
		"instanceType": node.Labels[k8slabels.LabelK8sNodeInstanceType],
	}
}

func ExtractNodeCommonTagsTo(node *v1.Node, to map[string]string) map[string]string {
	to["name"] = node.Name
	to["hostname"] = k8smetaextractor.PodMetaServiceInstance.NodeHostname(node)
	to["region"] = k8slabels.GetRegion(node.Labels)
	to["zone"] = k8slabels.GetZone(node.Labels)
	to["os"] = node.Labels[k8slabels.LabelK8sOs]
	to["arch"] = node.Labels[k8slabels.LabelK8sArch]
	to["instanceType"] = node.Labels[k8slabels.LabelK8sNodeInstanceType]
	return to
}

// SuppressCommonTags removes some extra ccommon tags
func SuppressCommonTags(tags map[string]string) {
	for _, key := range appconfig.StdAgentConfig.Data.Metric.SuppressedTags {
		delete(tags, key)
	}
}
