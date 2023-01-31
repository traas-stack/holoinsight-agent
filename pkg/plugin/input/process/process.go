package process

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"github.com/shirou/gopsutil/v3/process"
)

// TODO 问题 系统指标是否作为一个指标存储 还是 作为多个指标(开源倾向)
// 所有系统指标作为一个指标存储的问题是 整个指标太大了, 并且有一些指标(比如分区使用率)要压缩成一个字符串去传输, 否则分区路径应该是作为一个tags的
// TODO 代码重构

type (
	processInput struct {
		input.BaseInput
	}
)

func (i *processInput) Collect(ctx *input.CollectContext) ([]*model.DetailData, error) {
	d := model.NewDetailData()

	if util.IsLinux() {
		// 这个只需花1ms 反正是linux上特有的 这个无所谓
		pids, err := process.Pids()
		if err != nil {
			logger.Errorf("get pids error %+v", err)
			// 要填个0进去
			d.Values["process_pids"] = 0
			d.Values["process_threads"] = 0
		} else {
			totalThreads := readTotalThreads(pids)
			d.Values["process_pids"] = len(pids)
			d.Values["process_threads"] = totalThreads
		}
	} else {
		d.Values["process_pids"] = 0
		d.Values["process_threads"] = 0
	}

	return model.MakeDetailDataSlice(d), nil
}
