/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package extractor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8sutils"
	v1 "k8s.io/api/core/v1"
	"strings"
)

type (
	CriMetaService interface {
		// 提取应用名
		ExtractApp(pod *v1.Pod) string
		// 提取 hostname
		ExtractHostname(pod *v1.Pod) string
		// 提取主要容器名
		ExtractMainContainer(pod *v1.Pod) string
		// 检查是否是一个
		IsSidecar(container *cri.Container) bool
		// 是否是一个sandbox
		IsSandbox(container *cri.Container) bool
	}
	PodValueExtractFunc  func(pod *v1.Pod) string
	NodeValueExtractFunc func(node *v1.Node) string
	SandboxCheckHook     func(k8sContainerName string, labels map[string]string) bool

	Env struct {
		Name  string
		Value string
	}
	LabelBasedSandboxCheckHook struct {
		matchAny map[string]string
	}
)

func CreateLabelBasedSandboxCheckHook(matchAny map[string]string) SandboxCheckHook {
	return func(k8sContainerName string, labels map[string]string) bool {
		for k, v := range matchAny {
			if v2, ok := labels[k]; ok && v == v2 {
				return true
			}
		}

		return k8sutils.IsSandbox("", k8sContainerName, labels)
	}
}

// expr形如 k8slabel:app,env:EDAS_APP_NAME
func CreateLabelBasedPodMetaExtractFunc(expr string) PodValueExtractFunc {
	ss1 := strings.Split(expr, ",")
	var funcs []PodValueExtractFunc
	for _, pair := range ss1 {
		ss2 := strings.Split(pair, ":")
		refType := ss2[0]
		key := ss2[1]
		switch refType {
		case "k8slabel":
			fallthrough
		case "label":
			funcs = append(funcs, func(pod *v1.Pod) string {
				return pod.Labels[key]
			})
		case "env":
			// 必须是常量
			funcs = append(funcs, func(pod *v1.Pod) string {
				for i := range pod.Spec.Containers {
					container := &pod.Spec.Containers[i]
					for j := range container.Env {
						env := &container.Env[j]
						if env.Name == key {
							return env.Value
						}
					}
				}
				return ""
			})
		}
	}
	return Composite(funcs...)
}

func CreateLabelBasedNodeMetaExtractFunc(expr string) NodeValueExtractFunc {
	ss1 := strings.Split(expr, ",")
	var funcs []NodeValueExtractFunc
	for _, pair := range ss1 {
		ss2 := strings.Split(pair, ":")
		refType := ss2[0]
		key := ss2[1]
		switch refType {
		case "k8slabel":
			fallthrough
		case "label":
			funcs = append(funcs, func(node *v1.Node) string {
				return node.Labels[key]
			})
		}
	}
	return NodeComposite(funcs...)
}

func Composite(args ...PodValueExtractFunc) PodValueExtractFunc {
	return func(pod *v1.Pod) string {
		for _, f := range args {
			s := f(pod)
			if s != "" {
				return s
			}
		}
		return ""
	}
}

func NodeComposite(args ...NodeValueExtractFunc) NodeValueExtractFunc {
	return func(node *v1.Node) string {
		for _, f := range args {
			s := f(node)
			if s != "" {
				return s
			}
		}
		return ""
	}
}
