/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package standard

import (
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
)

type (
	Output struct {
		Tenant string
		O      output.Output
	}
)

func (o *Output) Write(metrics []*model.Metric) {
	oe := output.Extension{
		Tenant: o.Tenant,
	}
	o.O.WriteMetricsV1(metrics, oe)
}
