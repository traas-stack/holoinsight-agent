/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"encoding/json"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"net/http"
)

func (e *defaultCri) registerHttpHandlers() {
	http.HandleFunc("/api/meta/local/get", func(writer http.ResponseWriter, request *http.Request) {
		ns := request.URL.Query().Get("ns")
		podName := request.URL.Query().Get("pod")

		pod, err := e.GetPod(ns, podName)
		if err != nil {
			writer.Write([]byte("no found"))
			return
		}

		json.NewEncoder(writer).Encode(pod)
	})
	http.HandleFunc("/api/meta/local/getByCid", func(writer http.ResponseWriter, request *http.Request) {
		cid := request.URL.Query().Get("cid")
		container, ok := e.GetContainerByCid(cid)
		if !ok {
			writer.Write([]byte("no found"))
			return
		}

		json.NewEncoder(writer).Encode(container)
	})
	http.HandleFunc("/api/meta/local/list", func(writer http.ResponseWriter, request *http.Request) {
		state := e.state
		var ret []interface{}
		for _, pod := range state.pods {
			ret = append(ret, []string{pod.Namespace, pod.Name})
		}
		json.NewEncoder(writer).Encode(ret)
	})
	http.HandleFunc("/api/meta/local/hackErrorContainers", func(writer http.ResponseWriter, request *http.Request) {
		state := e.state
		ret := make([]string, 0)
		for _, pod := range state.pods {
			for _, container := range pod.All {
				if container.Sandbox {
					continue
				}
				if container.Hacked == 1 || container.Hacked == 5 {
					ret = append(ret, container.ShortContainerID())
				}
			}
		}
		json.NewEncoder(writer).Encode(ret)
	})

	http.HandleFunc("/api/cri/get", func(writer http.ResponseWriter, request *http.Request) {
		var pods []*cri.Pod

		if len(pods) == 0 {
			ip := request.URL.Query().Get("ip")
			if ip != "" {
				pods := criutils.FindPodsByIp(ioc.Crii, ip)
				json.NewEncoder(writer).Encode(pods)
				return
			}
		}

		if len(pods) == 0 {
			ns := request.URL.Query().Get("ns")
			podName := request.URL.Query().Get("pod")
			if pod, err := e.GetPod(ns, podName); err == nil {
				pods = []*cri.Pod{pod}
			}
		}

		if len(pods) == 0 {
			cid := request.URL.Query().Get("cid")
			if c, ok := e.GetContainerByCid(cid); ok {
				pods = []*cri.Pod{c.Pod}
			}
		}

		if len(pods) > 0 {
			json.NewEncoder(writer).Encode(pods)
			return
		}

		writer.WriteHeader(http.StatusNotFound)
		writer.Write([]byte("not found"))
	})
}
