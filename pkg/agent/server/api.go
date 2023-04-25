/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package server

import (
	"net/http"
	"sync"
)

type (
	ApiServerFunc struct {
		F          func(http.ResponseWriter, *http.Request)
		MoreUsages []string
	}
)

var (
	apiHandleFuncStore   = map[string]ApiServerFunc{}
	apiHandleFuncStoreMu sync.RWMutex
	apiHandleFuncMux     = http.DefaultServeMux
)

func RegisterApiHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), additionalUsages ...string) {
	apiHandleFuncStoreMu.Lock()
	if _, exist := apiHandleFuncStore[pattern]; !exist {
		apiHandleFuncStore[pattern] = ApiServerFunc{
			F:          handler,
			MoreUsages: additionalUsages,
		}
		apiHandleFuncMux.HandleFunc(pattern, handler)
	}
	apiHandleFuncStoreMu.Unlock()
}
