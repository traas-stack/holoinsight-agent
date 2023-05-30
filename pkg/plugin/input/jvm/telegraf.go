/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package jvm

import (
	"context"
	"errors"
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/standard/providers"
	"github.com/xin053/hsperfdata"
	"go.uber.org/zap"
	"path/filepath"
	"time"
)

const (
	measurement         = "jvm"
	perfDataFilePattern = "/tmp/hsperfdata_*/*"
	defaultOpTimeout    = 3 * time.Second
)

// TODO This file couples too many cri-related details and needs to be refactored later.

type (
	telegrafInput struct {
		state *jvmState
		task  *collecttask.CollectTask
	}
	perfDataReader func(path string) (map[string]interface{}, error)
)

var (
	multipleJavaProcessesErr = errors.New("[jvm] find multiple java processes")
)

func init() {
	providers.Register("jvmtask", NewTelegrafJvmInput)
}

func NewTelegrafJvmInput(target *collecttask.CollectTask) (api.Input, error) {
	return &telegrafInput{
		task: target,
	}, nil
}

func (i *telegrafInput) GetDefaultPrefix() string {
	return ""
}

func (i *telegrafInput) getProcessInfo(pid int32) (*criutils.ProcessInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
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

	biz, err := criutils.GetMainBizContainerE(ioc.Crii, i.task.Target.GetNamespace(), i.task.Target.GetPodName())
	if err != nil {
		return nil, err
	}
	return criutils.GetProcessInfo(ctx, ioc.Crii, biz, int(pid))
}

func (i *telegrafInput) getPerfDataPaths() (map[string]string, perfDataReader, error) {
	if !i.task.Target.IsTypePod() {
		if pathMap, err := hsperfdata.AllPerfDataPaths(); err != nil {
			return nil, nil, err
		} else {
			return pathMap, defaultReadPerfDataImpl, nil
		}
	}

	c, err := criutils.GetMainBizContainerE(ioc.Crii, i.task.Target.GetNamespace(), i.task.Target.GetPodName())
	if err != nil {
		return nil, nil, err
	}

	var paths []string
	var readPerfData perfDataReader
	if c.Runtime == cri.Runc {
		paths, err = filepath.Glob(filepath.Join(c.MergedDir, perfDataFilePattern))
		readPerfData = defaultReadPerfDataImpl
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
		defer cancel()
		paths, err = criutils.Glob(ctx, ioc.Crii, c, perfDataFilePattern)
		readPerfData = readPerfDataUsingDockerExec(c)
	}

	if err != nil {
		return nil, nil, err
	}

	pathMap := make(map[string]string, len(paths))
	for _, path := range paths {
		pid := filepath.Base(path)
		pathMap[pid] = path
	}
	return pathMap, readPerfData, nil
}

func (i *telegrafInput) Collect(a api.Accumulator) error {
	oldState := i.state
	newState := &jvmState{
		byPid: make(map[string]*pidJvmState),
	}
	i.state = newState

	pathMap, readFunc, err := i.getPerfDataPaths()
	if err != nil {
		return err
	}

	logger.Debugz("[jvm] resolve perf data paths", zap.Any("target", i.task.Target), zap.Any("paths", pathMap))

	// TODO how to known which one is the main java process?
	if len(pathMap) > 1 {
		return multipleJavaProcessesErr
	}

	for pid, perfPath := range pathMap {
		pid32, err := cast.ToInt32E(pid)
		if err != nil {
			logger.Errorz("[jvm] parse pid error", zap.Any("target", i.task.Target), zap.String("pid", pid), zap.Error(err))
			continue
		}
		javaProcess, err := i.getProcessInfo(pid32)
		if err != nil {
			logger.Debugz("[jvm] get process info error", zap.Any("target", i.task.Target), zap.String("pid", pid), zap.Error(err))
		}

		perfData, err := readFunc(perfPath)
		if err != nil {
			logger.Errorz("[jvm] read jvm perf data error", zap.Any("target", i.task.Target), zap.Int32("pid", pid32), zap.Error(err))
			continue
		}

		rawMetrics := make(map[string]interface{})
		tags := map[string]string{}

		addJvmMetrics(perfData, rawMetrics)
		addJvmMetricsFromProcess(javaProcess, rawMetrics, tags)

		newState.byPid[pid] = &pidJvmState{rawMetrics: rawMetrics}

		var lastPidState *pidJvmState
		if oldState != nil {
			lastPidState = oldState.byPid[pid]
		}
		finalMetrics := calcFinalMetrics(rawMetrics, lastPidState)

		for k, v := range finalMetrics {
			a.AddMetric(&model.Metric{
				Name:  fmt.Sprintf("jvm_%s", k),
				Tags:  tags,
				Value: cast.ToFloat64(v),
			})
		}
	}

	return nil
}

func defaultReadPerfDataImpl(path string) (map[string]interface{}, error) {
	return hsperfdata.ReadPerfData(path, true)
}

func readPerfDataUsingDockerExec(c *cri.Container) perfDataReader {
	return func(path string) (map[string]interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
		defer cancel()
		if tempPath, deleteFunc, err := criutils.CopyFromContainerToTempFile(ctx, ioc.Crii, c, path); err != nil {
			return nil, err
		} else {
			defer deleteFunc()
			return hsperfdata.ReadPerfData(tempPath, true)
		}
	}
}
