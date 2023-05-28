/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cpu

import (
	"bufio"
	"github.com/shirou/gopsutil/v3/cpu"
	"os"
	"strconv"
	"strings"
)

type (
	CpuTimeCsStat struct {
		M []CpuTimesStatExt
	}
	CpuTimesStatExt struct {
		cpu.TimesStat
		Total float64
		// gopsutil 内部概念
		Busy float64
		// vsar 特有
		Util float64
	}
)

func getCpuTimes() (*CpuTimeCsStat, error) {
	times, err := cpu.Times(false)
	if err != nil {
		return nil, err
	}
	m := make([]CpuTimesStatExt, len(times))
	for i := range times {
		t := &times[i]
		// irq 即 hirq
		total := t.User + t.System + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal + t.Idle
		m[i] = CpuTimesStatExt{
			TimesStat: *t,
			Total:     total,
			// see vendor/github.com/shirou/gopsutil/v3/cpu/cpu.go:101 getAllBusy
			Busy: total - t.Idle,
			Util: GetCpuUtil(t),
		}
	}
	return &CpuTimeCsStat{
		M: m,
	}, nil
}

// linux only
func readCpuStats() (ctxt int64, hirq int64, sirq int64, err error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ctxt") {
			sub := strings.SplitN(line, " ", 3)
			if len(sub) > 1 {
				ctxt, _ = strconv.ParseInt(sub[1], 10, 64)
			}
		} else if strings.HasPrefix(line, "intr") {
			sub := strings.SplitN(line, " ", 3)
			if len(sub) > 1 {
				hirq, _ = strconv.ParseInt(sub[1], 10, 64)
			}
		} else if strings.HasPrefix(line, "softirq") {
			sub := strings.SplitN(line, " ", 3)
			if len(sub) > 1 {
				sirq, _ = strconv.ParseInt(sub[1], 10, 64)
			}
		}
	}
	return ctxt, hirq, sirq, nil
}

func (st *CpuTimeCsStat) Gauge(other *CpuTimeCsStat) map[string]CpuTimesStatExt {
	size := len(other.M)
	merged := make(map[string]CpuTimesStatExt, size)

	for i := range other.M {
		ts2 := other.M[i]
		ts1 := st.M[i]

		// 得到总时间增量
		totalDelta := ts2.Total - ts1.Total
		if totalDelta <= 0 {
			continue
		}

		// https://psutil.readthedocs.io/en/latest/#cpu
		// CPU times are always supposed to increase over time, or at least remain the same, and that’s because time cannot go backwards.

		ts := CpuTimesStatExt{}
		ts.CPU = ts2.CPU

		// user: time spent by normal processes executing in user mode; on Linux this also includes guest time
		ts.User = 100 * (ts2.User - ts1.User - (ts2.Guest - ts1.Guest)) / totalDelta
		ts.System = 100 * (ts2.System - ts1.System) / totalDelta
		// same as user
		ts.Nice = 100 * (ts2.Nice - ts1.Nice - (ts2.GuestNice - ts1.GuestNice)) / totalDelta
		ts.Iowait = 100 * (ts2.Iowait - ts1.Iowait) / totalDelta
		ts.Irq = 100 * (ts2.Irq - ts1.Irq) / totalDelta
		ts.Softirq = 100 * (ts2.Softirq - ts1.Softirq) / totalDelta
		ts.Steal = 100 * (ts2.Steal - ts1.Steal) / totalDelta
		ts.Guest = 100 * (ts2.Guest - ts1.Guest) / totalDelta
		ts.GuestNice = 100 * (ts2.GuestNice - ts1.GuestNice) / totalDelta
		ts.Total = 100 * (ts2.Total - ts1.Total) / totalDelta
		// psutil builtin busy percent
		ts.Busy = 100 * (ts2.Busy - ts1.Busy) / totalDelta
		// Ant-Group Asar util percent
		ts.Util = 100 * (ts2.Util - ts1.Util) / totalDelta
		// TODO windows case
		merged[ts.CPU] = ts

	}
	return merged
}

func GetCpuTotal(t *cpu.TimesStat) float64 {
	return t.User + t.System + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal + t.Idle
}

func GetCpuBusy(t *cpu.TimesStat) float64 {
	return GetCpuTotal(t) - t.Idle
}

func GetCpuUtil(t *cpu.TimesStat) float64 {
	return t.User + t.System + t.Nice + t.Guest + t.Irq + t.Softirq
}
