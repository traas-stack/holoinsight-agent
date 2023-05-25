/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8ssysmetrics

import (
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics/cadvisor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics/common"
	"os"
	"time"
)

var providers = map[string]common.SysCollectorProvider{}

func init() {
	RegisterPodSystemResourceCollector("cadvisor", func(suffix string, interval time.Duration) (common.SysCollector, error) {
		return cadvisor.NewPodSystemResourceCollector(ioc.K8smm, ioc.Crii, suffix, interval), nil
	})
}

func RegisterPodSystemResourceCollector(name string, provider common.SysCollectorProvider) {
	providers[name] = provider
}

func NewPodSystemResourceCollector(suffix string, interval time.Duration) (common.SysCollector, error) {

	k8sSysCollector := os.Getenv("HI_K8S_SYS_COLLECTOR")

	if p, ok := providers[k8sSysCollector]; ok {
		return p(suffix, interval)
	}

	return providers["cadvisor"](suffix, interval)
}
