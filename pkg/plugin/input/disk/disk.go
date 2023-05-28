/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package disk

import (
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
)

type (
	diskInput struct {
	}
)

func (i *diskInput) GetDefaultPrefix() string {
	return ""
}

func (i *diskInput) Collect(a api.Accumulator) error {
	values := make(map[string]interface{})
	partitions, err := disk.Partitions(false)
	if err != nil {
		return err
	}
	diskTotal := uint64(0)
	diskUsed := uint64(0)
	for i := range partitions {
		p := &partitions[i]
		u, err := disk.Usage(p.Mountpoint)
		if err != nil {
			logger.Errorf("get usage error %+v", err)
			continue
		} else {
			diskTotal += u.Total
			diskUsed += u.Used
		}
	}
	if diskTotal > 0 {
		values["disk_total"] = diskTotal
		values["disk_used"] = diskUsed
		values["disk_util"] = 100 * float64(diskUsed) / float64(diskTotal)
	} else {
		values["disk_total"] = 0
		values["disk_used"] = 0
		values["disk_util"] = 0
	}

	input.AddMetrics(a, values)
	return nil
}
