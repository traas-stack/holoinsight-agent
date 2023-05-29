/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package standard

import (
	"encoding/json"
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/integration/base"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/standard"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
)

func ParsePipeline(task *collecttask.CollectTask) (*Pipeline, error) {
	i, err := standard.ParseInput(task)
	if err != nil {
		return nil, err
	}
	if i == nil {
		return nil, errors.New("ParseInput returns nil")
	}
	var out output.Output
	if appconfig.IsDev() {
		out, err = output.Parse("console", nil)
	} else {
		out, err = output.Parse("gateway", nil)
	}
	if err != nil {
		return nil, err
	}

	baseConf := &base.Conf{}
	if err := json.Unmarshal(task.Config.Content, baseConf); err != nil {
		return nil, err
	}

	tenant := task.Target.GetTenant()
	to := &Output{
		Tenant: tenant,
		O:      out,
	}

	return NewPipeline(task, baseConf, i, to)
}
