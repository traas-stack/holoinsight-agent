package inspect

import (
	"encoding/json"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/traas-stack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/nvidia_smi"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"os"
	"runtime"
	"time"
)

var uptime = time.Now()

func CreateAgentInfo() map[string]interface{} {
	return map[string]interface{}{
		"id":      agentmeta.GetAgentId(),
		"version": appconfig.StdAgentConfig.Version,
		"uptime":  uptime.Format(time.RFC3339),
	}
}

func Inspect(req *pb.InspectRequest, resp *pb.InspectResponse) error {
	m := make(map[string]interface{})
	info, _ := host.Info()
	system, role, _ := host.Virtualization()

	envTz := os.Getenv("TZ")
	tz, _ := time.Now().Zone()
	if agentInfoInEnv := os.Getenv("AGENT_INFO"); agentInfoInEnv != "" {
		agentInfo := make(map[string]interface{})
		json.Unmarshal([]byte(agentInfoInEnv), &agentInfo)
		m["agent"] = agentInfo
	} else {
		m["agent"] = CreateAgentInfo()
		if nvidia_smi.IsNvidiaEnabled() {
			if info, err := (&nvidia_smi.NvidiaInput{}).GetBasicInfo(); err == nil {
				m["gpu"] = info
			}
		}
	}
	m["host"] = map[string]interface{}{
		"info": info,
		"virtualization": map[string]interface{}{
			"system": system,
			"role":   role,
		},
		"timezone": tz,
		"envTz":    envTz,
	}

	m["golang"] = map[string]interface{}{
		"version": runtime.Version(),
		"name":    runtime.GOOS,
		"arch":    runtime.GOARCH,
	}
	{
		// cpu
		cpuCount, _ := cpu.Counts(true)

		// bool includeCpuInfos
		cpuInfos, _ := cpu.Info()

		m["cpu"] = map[string]interface{}{
			"count":    cpuCount,
			"cpuInfos": cpuInfos,
		}
	}

	{
		// mem
		if vm, err := mem.VirtualMemory(); err == nil {
			m["mem"] = vm
		}
	}

	{
		// disk
		if partitions, err := disk.Partitions(false); err == nil {
			d := map[string]interface{}{}
			m["disk"] = d
			for i := range partitions {
				p := &partitions[i]
				usage, err := disk.Usage(p.Mountpoint)
				if err != nil {
					continue
				}
				d[p.Mountpoint] = map[string]interface{}{
					"info":  p,
					"usage": usage,
				}
			}
		}
	}

	{
		// net
		// net.IOCounters()
		d := map[string]interface{}{}
		m["net"] = d
		d["ip"] = util.GetLocalIp()
		d["hostname"] = util.GetHostname()
		if list, err := net.Interfaces(); err == nil {
			normalInterfaces := make([]net.InterfaceStat, 0)
			for _, stat := range list {
				if util.StringSliceContains(stat.Flags, "loopback") {
					continue
				}
				normalInterfaces = append(normalInterfaces, stat)
			}
			d["interfaces"] = normalInterfaces
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	resp.Result = string(b)

	return nil
}
