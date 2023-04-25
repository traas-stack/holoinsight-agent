/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package api

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
)

type (
	Pipeline interface {
		Start()
		Stop()
		SetupConsumer(st *SubTask) error
		Update(func(Pipeline))
		View(func(Pipeline))
	}
	PipelineUpdateAware interface {
		UpdateFrom(Pipeline)
	}
	SubTask struct {
		CT      *collecttask.CollectTask
		SqlTask *collectconfig.SQLTask
	}
)
