/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package thread

import (
	"context"
	"encoding/json"
	"github.com/spf13/cast"
	cm "github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
)

type (
	input struct {
		task *collecttask.CollectTask
	}
)

func (i *input) GetDefaultPrefix() string {
	return ""
}

func (i *input) Collect(a api.Accumulator) error {
	if !i.task.Target.IsTypePod() {
		return nil
	}

	biz, err := criutils.GetMainBizContainerE(ioc.Crii, i.task.Target.GetNamespace(), i.task.Target.GetPodName())
	if err != nil {
		return err
	}

	r, err := ioc.Crii.Exec(context.Background(), biz, cri.ExecRequest{
		Cmd: []string{core.HelperToolPath, "countThread"},
	})
	if err != nil {
		return err
	}
	resp := &cm.Resp{}
	if err := json.NewDecoder(r.Stdout).Decode(resp); err != nil {
		return err
	}
	data := resp.Data.(map[string]interface{})
	for metric, value := range data {
		a.AddMetric(&model.Metric{
			Name:  metric,
			Tags:  make(map[string]string),
			Value: cast.ToFloat64(value),
		})
	}
	return nil
}
