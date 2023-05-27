/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"encoding/json"
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
}
