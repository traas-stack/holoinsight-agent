/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package load

import (
	"github.com/shirou/gopsutil/v3/load"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
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
