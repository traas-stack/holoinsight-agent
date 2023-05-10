/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package jvm

import (
	"context"
	"fmt"
	"github.com/influxdata/telegraf"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/telegraf/providers"
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

func (i *telegrafInput) SampleConfig() string {
	return ""
}

func (i *telegrafInput) getProcessInfo(pid int32) (*criutils.ProcessInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if !i.task.Target.IsTypePod() {
		p, err := process.NewProcessWithContext(ctx, pid)
		if err != nil {
			return nil, err
		}
		pi := &criutils.ProcessInfo{}
		pi.User, _ = p.Username()
		pi.Name, _ = p.Name()
		pi.CmdlineSlice, _ = p.CmdlineSlice()
		return pi, nil
	}

	namespace := i.task.Target.GetNamespace()
	podName := i.task.Target.GetPodName()
	pod, _ := ioc.Crii.GetPod(namespace, podName)
	if pod == nil {
		return nil, cri.NoPodError(namespace, podName)
	}

	return criutils.GetProcessInfo(ctx, ioc.Crii, pod.MainBiz(), int(pid))
}

func (i *telegrafInput) getPerfDataPaths() (map[string]string, func(path string) (map[string]interface{}, error), error) {
	if !i.task.Target.IsTypePod() {
		pathMap, err := hsperfdata.AllPerfDataPaths()
		if err != nil {
			return nil, nil, err
		}
		readPerfData := func(path string) (map[string]interface{}, error) {
			return hsperfdata.ReadPerfData(path, true)
		}
		return pathMap, readPerfData, nil
	}

	namespace := i.task.Target.GetNamespace()
	podName := i.task.Target.GetPodName()
	pod, _ := ioc.Crii.GetPod(namespace, podName)
	if pod == nil {
		return nil, nil, fmt.Errorf("fail to find pod [%s,%s]", namespace, podName)
	}

	c := pod.MainBiz()
	if c == nil {
		return nil, nil, cri.ErrMultiBiz
	}

	paths, err := criutils.Glob(context.Background(), ioc.Crii, c, "/tmp/hsperfdata_*/*")
	if err != nil {
		return nil, nil, err
	}

	pathMap := make(map[string]string)
	for _, path := range paths {
		pid := filepath.Base(path)
		pathMap[pid] = path
	}
	readPerfData := func(path string) (map[string]interface{}, error) {
		tempPath, deleteFunc, err := criutils.CopyFromContainerToTempFile(context.Background(), ioc.Crii, c, path)
		if err != nil {
			return nil, err
		}
		defer deleteFunc()

		return hsperfdata.ReadPerfData(tempPath, true)
	}
	return pathMap, readPerfData, nil
}

func (i *telegrafInput) Gather(a telegraf.Accumulator) error {
	oldState := i.state
	newState := &jvmState{
		byPid: make(map[string]*pidJvmState),
	}
	i.state = newState

	pathMap, readFunc, err := i.getPerfDataPaths()
	if err != nil {
		return err
	}

	if logger.DebugEnabled {
		logger.Debugz("[jvm] resolve perf data paths", zap.Any("target", i.task.Target), zap.Any("paths", pathMap))
	}

	for pid, perfPath := range pathMap {
		pid32, err := cast.ToInt32E(pid)
		if err != nil {
			logger.Errorz("[jvm] parse pid error", zap.Any("target", i.task.Target), zap.String("pid", pid), zap.Error(err))
			continue
		}
		javaProcess, err := i.getProcessInfo(pid32)
		if err != nil {
			// 这个是个可以失败的行为(预期内的), 所以记到debug里
			if logger.DebugEnabled {
				logger.Debugz("[jvm] get process info error", zap.Any("target", i.task.Target), zap.String("pid", pid), zap.Error(err))
			}
		}

		perfData, err := readFunc(perfPath)
		if err != nil {
			logger.Errorz("[jvm] fail to read jvm perf data", zap.Any("target", i.task.Target), zap.Int32("pid", pid32), zap.Error(err))
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
