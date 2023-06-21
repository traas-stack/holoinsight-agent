/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package jvm

import (
	"context"
	"encoding/gob"
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
	jvmInput struct {
		state *internalState
		task  *collecttask.CollectTask
	}
	perfDataReader func(path string) (map[string]interface{}, error)
)

func init() {
	gob.Register(&internalState{})
}

func (i *jvmInput) SaveState() (interface{}, error) {
	return i.state, nil
}

func (i *jvmInput) LoadState(state interface{}) error {
	i.state = state.(*internalState)
	return nil
}

var (
	multipleJavaProcessesErr = errors.New("[jvm] find multiple java processes")
)

func init() {
	providers.RegisterInputProvider("jvmtask", func(task *collecttask.CollectTask) (api.Input, error) {
		return &jvmInput{
			task: task,
			state: &internalState{
				ByPid: make(map[string]*pidJvmState),
			},
		}, nil
	})
}

func (i *jvmInput) GetDefaultPrefix() string {
	return ""
}

func (i *jvmInput) getProcessInfo(pid int32) (*criutils.ProcessInfo, error) {
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
		pi.Tgid, _ = p.Tgid()
		return pi, nil
	}

	biz, err := criutils.GetMainBizContainerE(ioc.Crii, i.task.Target.GetNamespace(), i.task.Target.GetPodName())
	if err != nil {
		return nil, err
	}
	return criutils.GetProcessInfo(ctx, ioc.Crii, biz, int(pid))
}

func (i *jvmInput) getPerfDataPaths() (map[string]string, perfDataReader, error) {
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
	pathsParsed := false
	if c.Runtime == cri.Runc {
		if hostPattern, err2 := cri.TransferToHostPathForContainer(c, perfDataFilePattern, false); err == nil {
			pathsParsed = true
			paths, err = filepath.Glob(hostPattern)
			readPerfData = defaultReadPerfDataImpl
		} else {
			err = err2
		}
	}
	// try another method
	if !pathsParsed {
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

func (i *jvmInput) Collect(a api.Accumulator) error {
	oldState := i.state
	newState := &internalState{
		ByPid: make(map[string]*pidJvmState),
	}
	i.state = newState

	pathMap, readFunc, err := i.getPerfDataPaths()
	if err != nil {
		return err
	}

	logger.Debugz("[jvm] resolve perf data paths", zap.Any("target", i.task.Target), zap.Any("paths", pathMap))

	hit := false
	for pid, perfPath := range pathMap {
		pid32, err := cast.ToInt32E(pid)
		if err != nil {
			logger.Errorz("[jvm] parse pid error", zap.Any("target", i.task.Target), zap.String("pid", pid), zap.Error(err))
			continue
		}
		javaProcess, err := i.getProcessInfo(pid32)
		if err != nil {
			logger.Errorz("[jvm] get process info error", zap.Any("target", i.task.Target), zap.String("pid", pid), zap.Error(err))
			continue
		}
		if javaProcess.Tgid != pid32 {
			logger.Errorz("[jvm] skip thread pid", zap.Any("target", i.task.Target), zap.String("pid", pid), zap.Error(err))
			continue
		}
		// When there has multi jvm processes, we only keeps one whose name is 'java'
		if len(pathMap) > 1 && javaProcess.Name != "java" {
			logger.Errorz("[jvm] ignore process", zap.Any("target", i.task.Target), zap.String("pid", pid), zap.Error(err))
			continue
		}
		perfData, err := readFunc(perfPath)
		if err != nil {
			logger.Errorz("[jvm] read jvm perf data error", zap.Any("target", i.task.Target), zap.Int32("pid", pid32), zap.Error(err))
			continue
		}
		if hit {
			return multipleJavaProcessesErr
		}
		hit = true
		rawMetrics := make(map[string]interface{})
		tags := map[string]string{}

		addJvmMetrics(perfData, rawMetrics)
		addJvmMetricsFromProcess(javaProcess, rawMetrics, tags)

		newState.ByPid[pid] = &pidJvmState{RawMetrics: rawMetrics}

		var lastPidState *pidJvmState
		if oldState != nil {
			lastPidState = oldState.ByPid[pid]
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
		if tempPath, deleteFunc, err := criutils.CopySmallFileFromContainerToTempFile(ctx, ioc.Crii, c, path); err != nil {
			return nil, err
		} else {
			defer deleteFunc()
			return hsperfdata.ReadPerfData(tempPath, true)
		}
	}
}
