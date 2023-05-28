/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package mem

import (
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
)

type (
	memInput struct {
	}
)

func (m *memInput) GetDefaultPrefix() string {
	return ""
}

func (m *memInput) Collect(a api.Accumulator) error {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return err

	}

	values := make(map[string]interface{})
	values["mem_total"] = vm.Total
	values["mem_used"] = vm.Used
	values["mem_avail"] = vm.Available
	values["mem_free"] = vm.Free
	values["mem_buff"] = vm.Buffers
	values["mem_cach"] = vm.Cached
	values["mem_dirty"] = vm.Dirty
	values["mem_util"] = vm.UsedPercent
	values["mem_rutil"] = 100.0 * float64(vm.Total-vm.Available) / float64(vm.Total)

	input.AddMetrics(a, values)
	return nil
}
