/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package jvm

import (
	"context"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/crihelper"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/telegraf/providers"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/influxdata/telegraf"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cast"
	"github.com/xin053/hsperfdata"
	"go.uber.org/zap"
	"path/filepath"
	"strings"
	"time"
)

const measurement = "jvm"

type (
	telegrafInput struct {
		state *jvmState
		task  *collecttask.CollectTask
	}
)

func init() {
	providers.Register("JvmTask", NewTelegrafJvmInput)
}

func NewTelegrafJvmInput(target *collecttask.CollectTask) (interface{}, error) {
	return &telegrafInput{
		task: target,
	}, nil
}

func (j *telegrafInput) SampleConfig() string {
	return ""
}

func (j *telegrafInput) getProcessInfo(pid int32) (*crihelper.ProcessInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if !j.task.Target.IsTypePod() {
		p, err := process.NewProcessWithContext(ctx, pid)
		if err != nil {
			return nil, err
		}
		pi := &crihelper.ProcessInfo{}
		pi.User, _ = p.Username()
		pi.Name, _ = p.Name()
		pi.CmdlineSlice, _ = p.CmdlineSlice()
		return pi, nil
	}

	namespace := j.task.Target.GetNamespace()
	podName := j.task.Target.GetPodName()
	pod, _ := ioc.Crii.GetPod(namespace, podName)
	if pod == nil {
		return nil, fmt.Errorf("no pod [%s,%s]", namespace, podName)
	}

	return crihelper.HelperProcesses(ctx, pod, int(pid))
}

func (j *telegrafInput) Gather(a telegraf.Accumulator) error {
	oldState := j.state
	newState := &jvmState{
		byPid: make(map[string]*pidJvmState),
	}
	j.state = newState

	var pathMap map[string]string
	// TODO 重构
	if j.task.Target.IsTypePod() {
		namespace := j.task.Target.GetNamespace()
		podName := j.task.Target.GetPodName()
		pod, _ := ioc.Crii.GetPod(namespace, podName)
		if pod == nil {
			return fmt.Errorf("fail to find pod [%s,%s]", namespace, podName)
		}
		// TODO 这里支持所有容器还是主容器?
		c := pod.MainBiz()
		if c == nil {
			return cri.ErrMultiBiz
		}
		if c.MergedDir == "" {
			return fmt.Errorf("invalid MergedDir, pod=%s", util.ToJsonString(pod))
		}

		// TODO 其实这个结果可以缓存一小段时间
		// TODO 这里别偶尔 mergeddir 逻辑
		dirsGlob := filepath.Join(c.MergedDir, "tmp", "hsperfdata_*", "*")
		paths, err := filepath.Glob(dirsGlob)
		if err != nil {
			return err
		}

		pathMap = make(map[string]string)
		for _, path := range paths {
			pid := filepath.Base(path)
			pathMap[pid] = path
		}
	} else {
		pathMap, _ = hsperfdata.AllPerfDataPaths()
	}

	if logger.DebugEnabled {
		logger.Debugz("[jvm] resolve perf data paths", zap.Any("target", j.task.Target), zap.Any("paths", pathMap))
	}

	for pid, perfPath := range pathMap {
		pid32, err := cast.ToInt32E(pid)
		if err != nil {
			logger.Errorz("[jvm] parse pid error", zap.Any("target", j.task.Target), zap.String("pid", pid), zap.Error(err))
			continue
		}
		javaProcess, err := j.getProcessInfo(pid32)
		if err != nil {
			// 这个是个可以失败的行为(预期内的), 所以记到debug里
			if logger.DebugEnabled {
				logger.Debugz("[jvm] get process info error", zap.Any("target", j.task.Target), zap.String("pid", pid), zap.Error(err))
			}
		}

		perfData, err := hsperfdata.ReadPerfData(perfPath, true)
		if err != nil {
			logger.Errorz("[jvm] fail to read jvm perf data", zap.Any("target", j.task.Target), zap.Int32("pid", pid32), zap.Error(err))
			continue
		}

		rawMetrics := make(map[string]interface{})
		tags := map[string]string{
			"pid": pid,
		}

		addJvmMetrics(perfData, rawMetrics)
		addJvmMetricsFromProcess(javaProcess, rawMetrics, tags)

		newState.byPid[pid] = &pidJvmState{rawMetrics: rawMetrics}

		// 有一些指标需要进行修正
		var lastPidState *pidJvmState
		if oldState != nil {
			lastPidState = oldState.byPid[pid]
		}
		finalMetrics := calcFinalMetrics(rawMetrics, lastPidState)

		a.AddFields(measurement, finalMetrics, tags)
	}

	return nil
}

func cmdlineContainsAny(cmdlineSlice []string, keywords ...string) bool {
	for _, s := range cmdlineSlice {
		s = strings.ToLower(s)
		for _, k := range keywords {
			if strings.Contains(s, k) {
				return true
			}
		}
	}
	return false
}
