package extractor

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri/dockerutils"
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri/pouch"
	"github.com/TRaaSStack/holoinsight-agent/pkg/k8s/k8slabels"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	v1 "k8s.io/api/core/v1"
	"strings"
)

type (
	DefaultPodMetaExtractor struct {
		App          PodValueExtractFunc
		Hostname     PodValueExtractFunc
		NodeHostname NodeValueExtractFunc
		Sandbox      SandboxCheckHook
		Sidecar      *DefaultSidecarCheckHook
	}
	DefaultSidecarCheckHook struct {
		Names []string
		Envs  []Env
	}
)

var (
	// 默认的 app 提取方式
	DefaultAppExtractorInstance = Composite(
		CreateLabelBasedPodMetaExtractFunc("label:app,label:app.kubernetes.io/name"))
	// 默认的 hostname 提取方式
	DefaultHostnameExtractorInstance = Composite(
		CreateLabelBasedPodMetaExtractFunc("label:HOSTNAME"),
		func(pod *v1.Pod) string { return util.FirstNotEmpty(pod.Spec.Hostname, pod.Name) })
	DefaultNodeHostnameExtractorInstance = func(node *v1.Node) string {
		return util.FirstNotEmpty(node.Labels[k8slabels.LabelK8sHostname], node.Name)
	}
	// 默认的 mainbiz 提取方式
	DefaultSidecarCheckHookInstance = NewDefaultSidecarCheckHook("name:sidecar")

	PodMetaServiceInstance = DefaultPodMetaExtractor{
		App:          DefaultAppExtractorInstance,
		Hostname:     DefaultHostnameExtractorInstance,
		NodeHostname: DefaultNodeHostnameExtractorInstance,
		Sidecar:      DefaultSidecarCheckHookInstance,
		Sandbox: CreateLabelBasedSandboxCheckHook(map[string]string{
			dockerutils.LabelDockerType: dockerutils.LabelValuePodSandbox,
			pouch.LabelPouchType:        pouch.LabelValueSandbox,
		}),
	}
	DefaultPodMetaService CriMetaService = &PodMetaServiceInstance
)

func NewDefaultSidecarCheckHook(sidecarExpr string) *DefaultSidecarCheckHook {

	var names []string
	var envs []Env

	ss1 := strings.Split(strings.TrimSpace(sidecarExpr), ",")
	for _, pair := range ss1 {
		ss2 := strings.Split(pair, ":")
		switch ss2[0] {
		case "env":
			if len(ss2) == 3 {
				envs = append(envs, Env{Name: ss2[1], Value: ss2[2]})
			}
		case "name":
			if len(ss2) == 2 {
				names = append(names, ss2[1])
			}
		}
	}

	return &DefaultSidecarCheckHook{
		Envs:  envs,
		Names: names,
	}
}

func (e *DefaultSidecarCheckHook) IsSandbox(container *cri.Container) bool {
	return container.Name == "" || dockerutils.IsSandbox(container.Labels) || pouch.IsSandbox(container.Labels)
}

func (e *DefaultSidecarCheckHook) IsSidecar(container *cri.Container) bool {
	pod := container.Pod
	if len(pod.Spec.Containers) == 0 {
		return false
	}
	if len(pod.Spec.Containers) == 1 {
		return false
	}
	if len(e.Names) > 0 {
		for _, keyword := range e.Names {
			if strings.Contains(container.Name, keyword) {
				return true
			}
		}
	}

	if len(e.Envs) > 0 {
		for _, env := range e.Envs {
			if containerEnv, ok := container.Env[env.Name]; ok && containerEnv == env.Value {
				return true
			}
		}
	}

	return false
}

func (e *DefaultSidecarCheckHook) Extract(pod *v1.Pod) string {
	if len(pod.Spec.Containers) == 0 {
		return ""
	}

	if len(pod.Spec.Containers) == 1 {
		return pod.Spec.Containers[0].Name
	}

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]

		// check names
		if len(e.Names) > 0 {
			isSidecar := false
			for _, keyword := range e.Names {
				if strings.Contains(container.Name, keyword) {
					isSidecar = true
					break
				}
			}

			if isSidecar {
				continue
			}
		}

		// check envs
		if len(e.Envs) > 0 {
			isSidecar := false

			for _, env := range e.Envs {
				for j := range container.Env {
					containerEnv := container.Env[j]
					if containerEnv.Name == env.Name && containerEnv.Value == env.Value {
						isSidecar = true
						break
					}
				}

				if isSidecar {
					break
				}
			}

			if isSidecar {
				continue
			}
		}

		return container.Name
	}

	// 那只能返回第一个容器的名字了
	return pod.Spec.Containers[0].Name
}

func (d *DefaultPodMetaExtractor) ExtractApp(pod *v1.Pod) string {
	return d.App(pod)
}

func (d *DefaultPodMetaExtractor) ExtractHostname(pod *v1.Pod) string {
	return d.Hostname(pod)
}

func (d *DefaultPodMetaExtractor) ExtractMainContainer(pod *v1.Pod) string {
	return d.Sidecar.Extract(pod)
}

func (d *DefaultPodMetaExtractor) IsSidecar(container *cri.Container) bool {
	return d.Sidecar.IsSidecar(container)
}

func (d *DefaultPodMetaExtractor) IsSandbox(container *cri.Container) bool {
	return d.Sandbox(container.Name, container.Labels)
}
