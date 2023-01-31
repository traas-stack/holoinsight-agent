package load

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input"
	"github.com/shirou/gopsutil/v3/load"
)

type (
	loadInput struct {
		input.BaseInput
	}
)

func (l *loadInput) Collect(ctx *input.CollectContext) ([]*model.DetailData, error) {
	d := model.NewDetailData()

	// TODO avg mis ???
	loadAvg, err := load.Avg()
	if err != nil {
		logger.Errorf("get load stat error %+v", err)
	} else {
		d.Values["load_load1"] = loadAvg.Load1
		d.Values["load_load5"] = loadAvg.Load5
		d.Values["load_load15"] = loadAvg.Load15
	}
	return model.MakeDetailDataSlice(d), nil
}
