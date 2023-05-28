/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package process

import (
	"github.com/shirou/gopsutil/v3/process"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
)

// TODO 问题 系统指标是否作为一个指标存储 还是 作为多个指标(开源倾向)
// 所有系统指标作为一个指标存储的问题是 整个指标太大了, 并且有一些指标(比如分区使用率)要压缩成一个字符串去传输, 否则分区路径应该是作为一个tags的
// TODO 代码重构

type (
	processInput struct {
	}
)

func (i *processInput) GetDefaultPrefix() string {
	return ""
}

func (i *processInput) Collect(a api.Accumulator) error {

	pids, err := process.Pids()
	if err != nil {
		return err
	}

	values := make(map[string]interface{})
	totalThreads := readTotalThreads(pids)
	values["process_pids"] = len(pids)
	values["process_threads"] = totalThreads

	input.AddMetrics(a, values)
	return nil
}
