/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package openmetric

import (
	"context"
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	v1 "k8s.io/api/core/v1"
	"strings"
	"sync"
)

type (
	dcgmConfig struct {
	}
	dcgmDiscovery struct {
		c     *dcgmConfig
		ctx   context.Context
		up    chan<- []*targetgroup.Group
		mutex sync.Mutex
		last  *v1.Pod
	}
)

func (o *dcgmDiscovery) OnAnyPodChanged() {
	o.once(false)
}

func (c *dcgmConfig) Name() string {
	return "dcgm"
}

func (c *dcgmConfig) NewDiscoverer(discovery.DiscovererOptions) (discovery.Discoverer, error) {
	return &dcgmDiscovery{c: c}, nil
}

func (o *dcgmDiscovery) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	o.ctx = ctx
	o.up = up
	ioc.Crii.AddListener(o)

	o.once(true)

	<-ctx.Done()
	ioc.Crii.RemoveListener(o)
}

func (p *dcgmDiscovery) buildPod(pod *v1.Pod) *targetgroup.Group {
	tg := &targetgroup.Group{
		Source: podSource(pod),
	}
	// PodIP can be empty when a pod is starting or has been evicted.
	if len(pod.Status.PodIP) == 0 {
		return tg
	}

	tg.Labels = podLabels(pod)
	tg.Labels[model.MetaLabelPrefix+"kubernetes_namespace"] = lv(pod.Namespace)
	tg.Labels[model.MetaLabelPrefix+"kubernetes_pod_name"] = lv(pod.Name)

	tg.Targets = append(tg.Targets, model.LabelSet{
		model.AddressLabel: lv(fmt.Sprintf("%s:%s", pod.Status.PodIP, "9400")),
	})

	return tg
}

func (o *dcgmDiscovery) once(init bool) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	pods := ioc.Crii.GetAllPods()
	var targetPod *cri.Pod
	for _, pod := range pods {
		if strings.Contains(pod.GetName(), "dcgm-export") {
			targetPod = pod
			break
		}
	}
	var tg *targetgroup.Group
	if targetPod != nil {
		if o.last != nil && o.last.Generation == targetPod.Generation {
			return
		}
		o.last = targetPod.Pod
		tg = o.buildPod(targetPod.Pod)
	} else {
		// delete
		if !init && o.last == nil || targetPod == nil {
			return
		}
		o.last = nil
		tg = &targetgroup.Group{
			Source: podSource(targetPod.Pod),
		}
	}
	o.up <- []*targetgroup.Group{tg}
}
