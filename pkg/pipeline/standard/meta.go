/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package standard

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/integration/base"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"strings"
)

func getPodMeta(task *collecttask.CollectTask, name string, pod *cri.Pod) (*cri.Pod, string) {
	if pod == nil && ioc.Crii != nil {
		pod, _ = ioc.Crii.GetPod(task.Target.GetNamespace(), task.Target.GetPodName())
	}
	if pod == nil {
		return pod, ""
	}
	if v, ok := pod.Labels[name]; ok && v != "" {
		return pod, v
	}
	if v, ok := pod.Annotations[name]; ok && v != "" {
		return pod, v
	}

	ss := strings.SplitN(name, ".", 2)
	if len(ss) != 2 {
		return pod, ""
	}
	subType := ss[0]
	subKey := ss[1]
	switch subType {
	case collectconfig.ElectRefMetaTypePodLabels:
		return pod, pod.Labels[subKey]
	case collectconfig.ElectRefMetaTypePodAnnotations:
		return pod, pod.Annotations[subKey]
	default:
		return pod, ""
	}
}

// createCommonTags creates common tags for this Pipeline
func createCommonTags(task *collecttask.CollectTask, baseConf *base.Conf) map[string]string {
	tags := make(map[string]string, len(baseConf.RefMetas))

	for key, item := range baseConf.RefMetas {
		if value, ok := task.Target.Meta[item.Name]; ok && value != "" {
			tags[key] = value
		}
	}

	switch task.Target.Type {
	case collecttask.TargetLocalhost:
		for key, item := range baseConf.RefMetas {
			if _, ok := tags[key]; ok {
				continue
			}
			if value := getLocalhostMeta(item.Name); value != "" {
				tags[key] = value
			}
		}
	case collecttask.TargetPod:
		var pod *cri.Pod
		var value string
		for key, item := range baseConf.RefMetas {
			if _, ok := tags[key]; ok {
				continue
			}
			if pod, value = getPodMeta(task, item.Name, pod); value != "" {
				tags[key] = value
			}
		}
	}

	meta.SuppressCommonTags(tags)
	return tags
}

func getLocalhostMeta(name string) string {
	var value string
	switch name {
	case "app":
		value = appconfig.StdAgentConfig.App
	case "ip":
		value = util.GetLocalIp()
	case "host":
		fallthrough
	case "hostname":
		value = util.GetHostname()
	}
	return value
}
