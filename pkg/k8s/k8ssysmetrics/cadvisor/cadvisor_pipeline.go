/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cadvisor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/standard/providers"
	"time"
)

func init() {
	providers.RegisterPipelineFactory("syscollector_cadvisor", func(task *collecttask.CollectTask) (api.Pipeline, error) {
		return k8ssysmetrics.NewSysCollectorPipeline("syscollector_cadvisor", NewPodSystemResourceCollector(ioc.Crii, "", time.Minute)), nil
	})
}
