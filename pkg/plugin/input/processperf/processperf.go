package processperf

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"
	"strconv"
	"strings"

	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input"
	"github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/cpu"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	gopsutil_cpu "github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
)

// 1. 进程需要有过滤功能
// 1.1 按用户名
// 1.2 按进程名 (简单名字或正则表达式)

type (
	processPerfInput struct {
		input.BaseInput
		includeUsernames map[string]struct{}
		excludeUsernames map[string]struct{}
		includeProcesses map[string]struct{}
		excludeProcesses map[string]struct{}
		includeKeywords  []string
		excludeKeywords  []string
		state            *processPerFState
	}
	processPerFState struct {
		cpu               *cpu.CpuState
		lastCpu           map[int32]*gopsutil_cpu.TimesStat
		lastIoCounterStat map[int32]*linuxIoStat
	}
)

func newProcessPerfInput(config0 input.Config) (*processPerfInput, error) {
	f := config0.(*collectconfig.From)
	cfg := f.ProcessPerf

	return &processPerfInput{
		includeUsernames: util.ConvertStringSliceToHashSet(cfg.IncludeUsernames),
		excludeUsernames: util.ConvertStringSliceToHashSet(cfg.ExcludeUsernames),
		includeProcesses: util.ConvertStringSliceToHashSet(cfg.IncludeProcesses),
		excludeProcesses: util.ConvertStringSliceToHashSet(cfg.ExcludeProcesses),
		includeKeywords:  cfg.IncludeKeywords,
		excludeKeywords:  cfg.ExcludeKeywords,
	}, nil
}

// 返回true则表示要处理该进程
func (i *processPerfInput) filter(p *process.Process) bool {
	if len(i.excludeUsernames) > 0 || len(i.includeUsernames) > 0 {
		username, err := p.Username()
		if err != nil {
			return false
		}
		if len(i.excludeUsernames) > 0 {
			if _, ok := i.excludeUsernames[username]; ok {
				return false
			}
		}
		if len(i.includeUsernames) > 0 {
			if _, ok := i.includeUsernames[username]; !ok {
				return false
			}
		}
	}
	if len(i.excludeProcesses) > 0 || len(i.includeProcesses) > 0 {
		name, err := p.Name()
		if err != nil {
			return false
		}
		if len(i.excludeProcesses) > 0 {
			if _, ok := i.excludeProcesses[name]; ok {
				return false
			}
		}
		if len(i.includeProcesses) > 0 {
			if _, ok := i.includeProcesses[name]; !ok {
				return false
			}
		}
	}
	if len(i.excludeKeywords) > 0 || len(i.includeKeywords) > 0 {
		cmd, err := p.Cmdline()
		if err != nil {
			return false
		}
		if len(i.excludeKeywords) > 0 {
			for _, k := range i.excludeKeywords {
				if strings.Contains(cmd, k) {
					return false
				}
			}
		}
		if len(i.includeKeywords) > 0 {
			matched := false
			for _, k := range i.excludeKeywords {
				if strings.Contains(cmd, k) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}
	return true
}

func (i *processPerfInput) Collect(ctx *input.CollectContext) ([]*model.DetailData, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("processperf collect error %+v", r)
		}
	}()

	state := i.state
	newState := &processPerFState{
		cpu:               &cpu.CpuState{},
		lastCpu:           make(map[int32]*gopsutil_cpu.TimesStat),
		lastIoCounterStat: make(map[int32]*linuxIoStat),
	}
	i.state = newState

	var totalDelta = float64(0)
	if cpuTimes, err := cpu.GetCpuTimes(); err == nil {
		newState.cpu.Cpu = cpuTimes
		if state != nil && state.cpu != nil {
			totalDelta = newState.cpu.Cpu.M[0].Total - state.cpu.Cpu.M[0].Total
		}
	}

	var result []*model.DetailData

	// 查所有pid
	pids, err := process.Pids()
	if err != nil {
		// TODO warn
		logger.Errorz("get process pids error", zap.Error(err))
		return nil, err
	}

	// TODO 这段代码效率极其低下
	// TODO 测速
processLoop:
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}

		if !i.filter(p) {
			continue processLoop
		}

		dd := model.NewDetailData()
		d := dd.Values
		result = append(result, dd)
		dd.Tags["pid"] = strconv.FormatInt(int64(p.Pid), 10)

		//d["Username"] = username
		//d ["Pid"] = p.Pid

		// TODO 看下代码 解析效率太低了 没缓存!
		//d["Name"], _ = p.Name()
		//d["Cmdline"], _ = p.Cmdline()

		d["num.threads"], _ = p.NumThreads()
		d["num.fds"], _ = p.NumFDs()

		if mi, err := p.MemoryInfo(); err == nil {
			d["mem.rss"] = mi.RSS
			// d["mem_shared"] = mex.Shared
		}
		d["mem.util"], _ = p.MemoryPercent()

		if processCpuTimes, err := p.Times(); err == nil {
			newState.lastCpu[pid] = processCpuTimes
			if state != nil && totalDelta > 0 {
				if oldProcessCpuTimes, ok := state.lastCpu[pid]; ok {

					d["cpu_user"] = 100 * (processCpuTimes.User - oldProcessCpuTimes.User - (processCpuTimes.Guest - oldProcessCpuTimes.Guest)) / totalDelta
					d["cpu_sys"] = 100 * (processCpuTimes.System - oldProcessCpuTimes.System) / totalDelta
					d["cpu_idle"] = 100 * (processCpuTimes.Idle - oldProcessCpuTimes.Idle) / totalDelta
					d["cpu_steal"] = 100 * (processCpuTimes.Steal - oldProcessCpuTimes.Steal) / totalDelta
					d["cpu_hirq"] = 100 * (processCpuTimes.Irq - oldProcessCpuTimes.Irq) / totalDelta
					d["cpu_sirq"] = 100 * (processCpuTimes.Softirq - oldProcessCpuTimes.Softirq) / totalDelta
					d["cpu_nice"] = 100 * (processCpuTimes.Nice - oldProcessCpuTimes.Nice - (processCpuTimes.GuestNice - oldProcessCpuTimes.GuestNice)) / totalDelta

					d["cpu_guest"] = 100 * (processCpuTimes.Guest - oldProcessCpuTimes.Guest) / totalDelta
					d["cpu_guestnice"] = 100 * (processCpuTimes.GuestNice - oldProcessCpuTimes.GuestNice) / totalDelta

					d["cpu_iowait"] = 100 * (processCpuTimes.Iowait - oldProcessCpuTimes.Iowait) / totalDelta
					cpuUtil := 100 * (cpu.GetCpuUtil(processCpuTimes) - cpu.GetCpuUtil(oldProcessCpuTimes)) / totalDelta
					d["cpu_util"] = cpuUtil
					d["cpu_busy"] = 100 * (cpu.GetCpuBusy(processCpuTimes) - cpu.GetCpuBusy(oldProcessCpuTimes)) / totalDelta

				}
			}
		}

		if ioStatus, err := readIoStatus(pid); err == nil {
			newState.lastIoCounterStat[pid] = ioStatus
			if state != nil {
				if lastIoCounters, ok := state.lastIoCounterStat[pid]; ok {
					d["io_rchar"] = ioStatus.rchar - lastIoCounters.rchar
					d["io_wchar"] = ioStatus.wchar - lastIoCounters.wchar
					// TODO ???
					d["io_read_bytes"] = ioStatus.read_bytes - lastIoCounters.read_bytes
					d["io_write_bytes"] = ioStatus.write_bytes - lastIoCounters.write_bytes
					d["io_syscr"] = ioStatus.syscr - lastIoCounters.syscr
					d["io_syscw"] = ioStatus.syscw - lastIoCounters.syscw
				}
			}
		}

		result = fillConnectionsMetrics(result, p, dd.Tags)

		if openFilesStats, err := p.OpenFiles(); err == nil {
			//files := 0
			//sockets := 0
			//anonInodes := 0
			//pipes := 0
			//nulls := 0
			//randoms := 0
			//other := 0
			count := map[string]int{}
			for i := range openFilesStats {
				openFilesStat := &openFilesStats[i]
				path := openFilesStat.Path
				var type2 string
				if strings.HasPrefix(path, "/") {
					type2 = "file"
				} else if strings.HasPrefix(path, "socket:") {
					type2 = "socket"
				} else if strings.HasPrefix(path, "anon_inode:") {
					type2 = "anon_inode"
				} else if strings.HasPrefix(path, "pipe:") {
					type2 = "pipe"
				} else if path == "/dev/null" {
					type2 = "null"
				} else if path == "/dev/random" || path == "/dev/urandom" {
					type2 = "random"
				} else {
					type2 = "other"
					logger.Infoz("openFilesStat", zap.Stringer("openFilesStat", openFilesStat))
				}
				count[type2] = count[type2] + 1
			}
			for type2, v := range count {
				tags := util.CopyStringMapCap(dd.Tags, len(dd.Tags)+1)
				tags["type"] = type2
				dd2 := model.NewDetailData().WithTags(tags).WithValue("fds", v)
				result = append(result, dd2)
			}
		}
	}

	// d["Connections"] = count

	// d["Connections"] = conns
	// d["NumCtxSwitches"], _ = p.NumCtxSwitches()

	//d["Background"], _ = p.Background()
	//d["Status"], _ = p.Status()

	// d["MemoryInfo"], _ = p.MemoryInfo()
	//d["IOnice"], _ = p.IOnice()
	//d["IsRunning"], _ = p.IsRunning()
	//d["Cwd"], _ = p.Cwd()
	//d["Foreground"], _ = p.Foreground()
	// d["Threads"], _ = p.Threads()
	//d["Rlimit"], _ = p.Rlimit()
	//d["RlimitUsage"], _ = p.RlimitUsage(true)
	//d["Exe"], _ = p.Exe()
	//d["Nice"], _ = p.Nice()
	//d["Groups"], _ = p.Groups()
	//d["Username"], _ = p.Username()
	//d["Gids"], _ = p.Gids()
	//d["Uids"], _ = p.Uids()
	//d["CPUAffinity"], _ = p.CPUAffinity()
	//d["CreateTime"], _ = p.CreateTime()
	//d["OpenFiles"], _ = p.OpenFiles()
	//d["PageFaults"], _ = p.PageFaults()
	// d["Percent"], _ = p.Percent(time.Second)
	//break
	return result, nil
}
