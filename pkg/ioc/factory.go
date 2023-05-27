/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

// Package ioc is a place for Inversion of Control / DI.
// But currently its implementation is very crude.
package ioc

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry"
	"k8s.io/client-go/kubernetes"
	"sync"
)

var Crii cri.Interface
var CollectTaskManager *collecttask.Manager
var RegistryService *registry.Service
var K8sClientset *kubernetes.Clientset

var m sync.Map

func Register(name string, obj interface{}) {
	m.Store(name, obj)
}

func Get(name string) (interface{}, bool) {
	return m.Load(name)
}

func MustGet(name string) (ret interface{}) {
	ret, _ = m.Load(name)
	return
}
