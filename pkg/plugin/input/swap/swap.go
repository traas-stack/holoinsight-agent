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
	swapInput struct {
	}
)

func (m *swapInput) GetDefaultPrefix() string {
	return ""
}

func (m *swapInput) Collect(a api.Accumulator) error {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	values := make(map[string]interface{})
	values["swap_total"] = vm.SwapTotal
	values["swap_free"] = vm.SwapFree
	values["swap_cach"] = vm.Cached

	input.AddMetrics(a, values)
	return nil
}
