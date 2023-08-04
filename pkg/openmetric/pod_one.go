/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package openmetric

import (
	"context"
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	_ "github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	v1 "k8s.io/api/core/v1"
	"sync"
	_ "unsafe"
)

type (
	onePodDiscoveryConfig struct {
		namespace string
		name      string
		port      string
	}
	onePodDiscovery struct {
		c     *onePodDiscoveryConfig
		ctx   context.Context
		up    chan<- []*targetgroup.Group
		mutex sync.Mutex
		last  *v1.Pod
	}
)

//go:linkname podLabels github.com/prometheus/prometheus/discovery/kubernetes.podLabels
func podLabels(*v1.Pod) model.LabelSet

//go:linkname lv github.com/prometheus/prometheus/discovery/kubernetes.lv
func lv(string) model.LabelValue

//go:linkname podSource github.com/prometheus/prometheus/discovery/kubernetes.podSource
func podSource(*v1.Pod) string

func (o *onePodDiscovery) OnAnyPodChanged() {
	o.once(false)
}

func (c *onePodDiscoveryConfig) Name() string {
	return "onepod"
}

func (c *onePodDiscoveryConfig) NewDiscoverer(discovery.DiscovererOptions) (discovery.Discoverer, error) {
	return &onePodDiscovery{c: c}, nil
}

func (o *onePodDiscovery) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	o.ctx = ctx
	o.up = up
	ioc.Crii.AddListener(o)

	o.once(true)

	<-ctx.Done()
	ioc.Crii.RemoveListener(o)
}

func (p *onePodDiscovery) buildPod(pod *v1.Pod) *targetgroup.Group {
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
		model.AddressLabel: lv(fmt.Sprintf("%s:%s", pod.Status.PodIP, p.c.port)),
	})

	return tg
}

func (o *onePodDiscovery) once(init bool) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	pod, err := ioc.Crii.GetPod(o.c.namespace, o.c.name)
	var tg *targetgroup.Group
	if err == nil {
		if o.last != nil && o.last.Generation == pod.Generation {
			return
		}
		o.last = pod.Pod
		tg = o.buildPod(pod.Pod)
	} else {
		// delete
		if !init && o.last == nil {
			return
		}
		o.last = nil
		tg = &targetgroup.Group{
			Source: podSource(pod.Pod),
		}
	}
	o.up <- []*targetgroup.Group{tg}
}
