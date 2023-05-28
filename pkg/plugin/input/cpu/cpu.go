/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cpu

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

// TODO 问题 系统指标是否作为一个指标存储 还是 作为多个指标(开源倾向)
// 所有系统指标作为一个指标存储的问题是 整个指标太大了, 并且有一些指标(比如分区使用率)要压缩成一个字符串去传输, 否则分区路径应该是作为一个tags的
// TODO 代码重构

type (
	cpuInput struct {
		state *CpuState
	}
	CpuState struct {
		// 上次执行时间
		Time int64
		Cpu  *CpuTimeCsStat
		Ctxt int64
		Sirq int64
		Hirq int64
	}
)

var (
	_ api.Input = &cpuInput{}
)

func (i *cpuInput) GetDefaultPrefix() string {
	return ""
}

func (i *cpuInput) Collect(a api.Accumulator) error {
	now := util.CurrentMS()

	state := i.state
	newState := &CpuState{
		Time: now,
	}
	i.state = newState

	mills := int64(0)
	if state != nil {
		mills = now - state.Time
	}

	values := make(map[string]interface{})

	if cpuCounts, err := cpu.Counts(true); err == nil {
		values["cpu_total_cores"] = cpuCounts
	}

	cpuTimesNow, err := getCpuTimes()
	newState.Cpu = cpuTimesNow
	if err == nil && state != nil && state.Cpu != nil {
		m := state.Cpu.Gauge(cpuTimesNow)
		cpuTimes, _ := m["cpu-total"]
		newState.Cpu = cpuTimesNow

		values["cpu_user"] = cpuTimes.User
		values["cpu_sys"] = cpuTimes.System
		values["cpu_idle"] = cpuTimes.Idle
		values["cpu_nice"] = cpuTimes.Nice
		values["cpu_iowait"] = cpuTimes.Iowait
		values["cpu_hirq"] = cpuTimes.Irq
		values["cpu_sirq"] = cpuTimes.Softirq
		values["cpu_steal"] = cpuTimes.Steal
		values["cpu_guest"] = cpuTimes.Guest
		values["cpu_guestnice"] = cpuTimes.GuestNice
		values["cpu_busy"] = cpuTimes.Busy
		values["cpu_util"] = cpuTimes.Util
	}

	ctxt, hirq, sirq, err := readCpuStats()
	newState.Ctxt = ctxt
	newState.Hirq = hirq
	newState.Sirq = sirq
	if err == nil && state != nil && state.Ctxt > 0 {
		values["cpu_counter_ctxt"] = (ctxt - state.Ctxt) * 1000 / mills
		values["cpu_counter_hirq"] = (hirq - state.Hirq) * 1000 / mills
		values["cpu_counter_sirq"] = (sirq - state.Sirq) * 1000 / mills
	}

	input.AddMetrics(a, values)
	return nil
}
