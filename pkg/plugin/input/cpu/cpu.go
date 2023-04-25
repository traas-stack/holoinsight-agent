/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cpu

import (
	"bytes"
	"encoding/gob"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

// TODO 问题 系统指标是否作为一个指标存储 还是 作为多个指标(开源倾向)
// 所有系统指标作为一个指标存储的问题是 整个指标太大了, 并且有一些指标(比如分区使用率)要压缩成一个字符串去传输, 否则分区路径应该是作为一个tags的
// TODO 代码重构

type (
	cpuInput struct {
		input.BaseInput
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

func (i *cpuInput) SerializeState() ([]byte, error) {
	b := bytes.NewBuffer(nil)
	err := gob.NewEncoder(b).Encode(i.state)
	return b.Bytes(), err
}

func (i *cpuInput) DeserializeState(in []byte) error {
	if in == nil {
		return nil
	}

	b := bytes.NewBuffer(in)
	newState := &CpuState{}
	i.state = newState
	return gob.NewDecoder(b).Decode(newState)
}

func (i *cpuInput) Collect(ctx *input.CollectContext) ([]*model.DetailData, error) {
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

	d := model.NewDetailData()

	// logical 是否包含超线程
	cpuCounts, err := cpu.Counts(true)
	if err != nil {
		logger.Warnf("get cpu counts error %+v", err)
	}
	d.Values["cpu_total_cores"] = cpuCounts

	cpuTimesNow, err := GetCpuTimes()
	if err != nil {
		return nil, err
	}
	newState.Cpu = cpuTimesNow

	if state != nil && state.Cpu != nil {
		m := state.Cpu.Gauge(cpuTimesNow)
		cpuTimes, _ := m["cpu-total"]
		newState.Cpu = cpuTimesNow

		// cpu 基本指标
		d.Values["cpu_user"] = cpuTimes.User
		d.Values["cpu_sys"] = cpuTimes.System
		d.Values["cpu_idle"] = cpuTimes.Idle
		d.Values["cpu_nice"] = cpuTimes.Nice
		d.Values["cpu_iowait"] = cpuTimes.Iowait
		d.Values["cpu_hirq"] = cpuTimes.Irq
		d.Values["cpu_sirq"] = cpuTimes.Softirq
		d.Values["cpu_steal"] = cpuTimes.Steal
		d.Values["cpu_guest"] = cpuTimes.Guest
		d.Values["cpu_guestnice"] = cpuTimes.GuestNice
		d.Values["cpu_busy"] = cpuTimes.Busy
		d.Values["cpu_util"] = cpuTimes.Util
	}

	if util.IsLinux() {
		ctxt, hirq, sirq, err := readCpuStats()
		if err != nil {
			logger.Warnf("get cpu ctxt error %+v", err)
		} else {
			newState.Ctxt = ctxt
			newState.Hirq = hirq
			newState.Sirq = sirq
			if state != nil && state.Ctxt > 0 {
				// 上下文切换次数
				d.Values["cpu_counter_ctxt"] = (ctxt - state.Ctxt) * 1000 / mills
				// 硬中断次数
				d.Values["cpu_counter_hirq"] = (hirq - state.Hirq) * 1000 / mills
				// 软中断次数
				d.Values["cpu_counter_sirq"] = (sirq - state.Sirq) * 1000 / mills
			}
		}
	}

	return model.MakeDetailDataSlice(d), nil
}
