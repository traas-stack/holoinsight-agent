/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8smeta

import (
	"encoding/json"
	"fmt"
	"github.com/jpillora/backoff"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"time"
)

type (
	// k8s 元数据管理器入口, 在我们代码里要获取跟k8s相关的元数据都从它来拿
	Manager struct {
		Clientset *kubernetes.Clientset
		PodMeta   *PodMeta
		NodeMeta  *NodeMeta
		LocalMeta *LocalMeta
	}
)

func NewManager(clientset *kubernetes.Clientset) *Manager {
	lm := &LocalMeta{}
	m := &Manager{
		Clientset: clientset,
		LocalMeta: lm,
		PodMeta:   newPodMeta(lm.NodeName(), clientset.CoreV1().RESTClient()),
		NodeMeta:  newNodeMeta(clientset.CoreV1().RESTClient()),
	}
	return m
}

func (m *Manager) Stop() {
	m.PodMeta.stop()
	m.NodeMeta.stop()
}

func (m *Manager) Start() {
	m.PodMeta.start()
	m.NodeMeta.start()

	controllers := []cache.Controller{m.PodMeta.informer, m.NodeMeta.informer}

	b := &backoff.Backoff{
		Factor: 2,
		Jitter: true,
		Min:    50 * time.Millisecond,
		Max:    time.Second,
	}

	// Or use k8s helper cache.WaitForCacheSync(nil, controller.HasSynced) ?
	for _, controller := range controllers {
		for !controller.HasSynced() {
			logger.Metaz("[k8s] [meta] wait meta sync")
			time.Sleep(b.Duration())
		}
	}

	m.registerHttpHandlers()
}

func (m *Manager) GetLocalHostPods() []*v1.Pod {
	return m.PodMeta.GetPodsByHostIP(m.LocalMeta.HostIP())
}

func (m *Manager) registerHttpHandlers() {
	// Query local pod info by namespace and podName
	http.HandleFunc("/api/meta/k8s/pods/get", func(writer http.ResponseWriter, request *http.Request) {
		ns := request.URL.Query().Get("ns")
		name := request.URL.Query().Get("name")
		pod := m.PodMeta.GetPodByName(ns, name)
		if pod == nil {
			writer.Write([]byte("not found"))
			return
		}
		json.NewEncoder(writer).Encode(pod)
	})

	// Query local pod info by podIP
	http.HandleFunc("/api/meta/k8s/pods/byIp", func(writer http.ResponseWriter, request *http.Request) {
		ip := request.URL.Query().Get("ip")
		pods := m.PodMeta.GetPodsByIP(ip)
		simple := m.convertToSimplePods(pods)
		json.NewEncoder(writer).Encode(simple)
	})

	// Query all local pod info
	http.HandleFunc("/api/meta/k8s/pods/local", func(writer http.ResponseWriter, request *http.Request) {
		pods := m.GetLocalHostPods()
		simple := m.convertToSimplePods(pods)
		json.NewEncoder(writer).Encode(simple)
	})

	// Query local pods by app
	http.HandleFunc("/api/meta/k8s/pods/byApp", func(writer http.ResponseWriter, request *http.Request) {
		namespace := request.URL.Query().Get("namespace")
		app := request.URL.Query().Get("app")
		pods := m.PodMeta.GetPodsByApp(namespace, app)
		simple := m.convertToSimplePods(pods)
		json.NewEncoder(writer).Encode(simple)
	})

	// nodes
	http.HandleFunc("/api/meta/k8s/nodes", func(writer http.ResponseWriter, request *http.Request) {
		json.NewEncoder(writer).Encode(m.NodeMeta.GetAllNodes())
	})
	http.HandleFunc("/api/meta/k8s/nodes/list", func(writer http.ResponseWriter, request *http.Request) {
		json.NewEncoder(writer).Encode(m.NodeMeta.GetAllNodes())
	})
}

func (m *Manager) convertToSimplePods(pods []*v1.Pod) []interface{} {
	var simple []interface{}

	for _, pod := range pods {
		simple = append(simple, m.convertToSimplePod(pod))
	}
	return simple
}

func (m *Manager) convertToSimplePod(pod *v1.Pod) map[string]interface{} {
	var containers []string
	for i, c := range pod.Spec.InitContainers {
		// 实际的cid能不能打印一下
		cs := pod.Status.InitContainerStatuses[i]
		containers = append(containers, fmt.Sprintf("i,%s,%s,%t", c.Name, cs.ContainerID, cs.Ready))
	}
	for i, c := range pod.Spec.Containers {
		cs := pod.Status.ContainerStatuses[i]
		containers = append(containers, fmt.Sprintf("c,%s,%s,%t", c.Name, cs.ContainerID, cs.Ready))
	}
	hostname := k8smetaextractor.PodMetaServiceInstance.ExtractHostname(pod)
	app := k8smetaextractor.DefaultPodMetaService.ExtractApp(pod)
	return map[string]interface{}{
		"namespace":  pod.Namespace,
		"name":       pod.Name,
		"ip":         pod.Status.PodIP,
		"hostname":   hostname,
		"hostIP":     pod.Status.HostIP,
		"phase":      pod.Status.Phase,
		"app":        app,
		"containers": containers,
	}
}
