/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package meta

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
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

	for _, key := range appconfig.StdAgentConfig.Data.Metric.SuppressedTags {
		delete(tags, key)
	}
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
