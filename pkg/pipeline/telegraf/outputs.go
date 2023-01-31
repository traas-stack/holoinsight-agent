package telegraf

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/plugin/output"
)

type (
	Output struct {
		Tenant string
		o      output.Output
	}
)

func (o *Output) Write(metrics []*model.Metric) {
	oe := output.Extension{
		Tenant: o.Tenant,
	}
	o.o.WriteMetrics(metrics, oe)
}
