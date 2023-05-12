/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package nvidia_smi

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"go.uber.org/zap"
	"os/exec"
	"strings"
)

type (
	NvidiaInput struct{}
	queryItem   struct {
		smiQueryGpu string
		metric      string
	}
)

// queryItems contain query info for nvidia-smi
var queryItems = []*queryItem{
	{
		smiQueryGpu: "utilization.gpu",
		metric:      "system_gpu_util",
	},
	{
		smiQueryGpu: "memory.used",
		metric:      "system_gpu_mem_used",
	},
	{
		smiQueryGpu: "memory.free",
		metric:      "system_gpu_mem_free",
	},
	{
		smiQueryGpu: "memory.total",
		metric:      "system_gpu_mem_total",
	},
	{
		smiQueryGpu: "utilization.memory",
		metric:      "system_gpu_mem_util",
	},
	{
		smiQueryGpu: "power.draw",
		metric:      "system_gpu_power_draw",
	},
	{
		smiQueryGpu: "temperature.gpu",
		metric:      "system_gpu_temperature",
	},
	{
		smiQueryGpu: "clocks.current.sm",
		metric:      "system_gpu_clocks_current_sm",
	},
}

// convertToFloat64 converts output string of nvidia-smi to float64
func convertToFloat64(item *queryItem, str string) (float64, error) {
	str = strings.TrimSpace(str)

	if strings.HasPrefix(item.smiQueryGpu, "memory.") {
		index := strings.LastIndexByte(str, ' ')
		if index < 0 {
			return 0, errors.New("invalid")
		}
		f64, err := cast.ToFloat64E(str[:index])
		if err != nil {
			return f64, err
		}
		switch str[index+1:] {
		case "MiB":
			f64 *= 1024 * 1024
		case "GiB":
			f64 *= 1024 * 1024 * 1024
		case "TiB":
			f64 *= 1024 * 1024 * 1024 * 1024
		default:
			return 0, nil
		}
		return f64, nil
	}
	if strings.HasSuffix(str, " %") {
		str = str[:len(str)-2]
		v, err := cast.ToFloat64E(str)
		return v / 100, err
	}
	if strings.HasSuffix(str, " W") {
		str = str[:len(str)-2]
	}
	if strings.HasSuffix(str, " MHz") {
		str = str[:len(str)-4]
	}
	return cast.ToFloat64E(str)
}

func (i *NvidiaInput) GetDefaultPrefix() string {
	return ""
}

func (n *NvidiaInput) Collect(a api.Accumulator) error {
	sb := strings.Builder{}
	sb.WriteString("--query-gpu=index")
	for _, item := range queryItems {
		sb.WriteByte(',')
		sb.WriteString(item.smiQueryGpu)
	}
	r, err := n.executeCommand(sb.String(), "--format=csv,noheader")
	if err != nil {
		return err
	}
	for _, ss := range r {
		tags := map[string]string{
			"index": ss[0],
		}
		for i, item := range queryItems {
			str := ss[1+i]
			f64, err := convertToFloat64(item, str)
			if err != nil {
				logger.Errorz("[input] [nvidia_smi] fail to convert", zap.Any("item", item), zap.String("str", str))
				continue
			}
			a.AddMetric(&model.Metric{
				Name:      item.metric,
				Tags:      tags,
				Timestamp: 0,
				Value:     f64,
			})
		}
	}
	a.AddMetric(&model.Metric{
		Name:      "system_gpu_count",
		Tags:      make(map[string]string),
		Timestamp: 0,
		Value:     float64(len(r)),
	})
	return nil
}

func (n *NvidiaInput) executeCommand(args ...string) ([][]string, error) {
	r, err := n.executeCommand0(args...)
	if err != nil {
		return nil, err
	}
	return csv.NewReader(r.Stdout).ReadAll()
}

func (n *NvidiaInput) executeCommand0(args ...string) (cri.ExecResult, error) {
	var cmd *exec.Cmd
	switch appconfig.StdAgentConfig.Mode {
	case core.AgentModeDaemonset:
		chrootArgs := make([]string, 0, 2+len(args))
		chrootArgs = append(chrootArgs, core.GetHostfs(), toolPath)
		chrootArgs = append(chrootArgs, args...)
		cmd = exec.Command("chroot", chrootArgs...)
	case core.AgentModeSidecar:
		cmd = exec.Command(toolPath, args...)
	default:
		return cri.ExecResult{ExitCode: -1}, errors.New("unsupported")
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()

	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return cri.ExecResult{ExitCode: cmd.ProcessState.ExitCode()}, errors.Wrapf(err, "%s exec fail, stdout=[%s] stderr=[%s]", toolPath, stdout.String(), stderr.String())
	}

	return cri.ExecResult{
		Cmd:      cmd.String(),
		ExitCode: cmd.ProcessState.ExitCode(),
		Stdout:   stdout,
		Stderr:   stderr,
	}, nil
}

func (n *NvidiaInput) GetBasicInfo() (interface{}, error) {
	r, err := n.executeCommand0("-L")
	if err != nil {
		return nil, nil
	} else {
		scanner := bufio.NewScanner(r.Stdout)
		scanner.Split(bufio.ScanLines)
		var gpus []string
		for scanner.Scan() {
			gpus = append(gpus, scanner.Text())
		}
		return map[string]interface{}{
			"count": len(gpus),
			"gpus":  gpus,
		}, nil
	}
}
