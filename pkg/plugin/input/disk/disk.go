/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package disk

import (
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	"github.com/shirou/gopsutil/v3/disk"
)

type (
	diskInput struct {
		state *diskState
		input.BaseInput
	}
	diskState struct {
		stats map[string]disk.IOCountersStat
	}
)

func (i *diskInput) Collect(ctx *input.CollectContext) ([]*model.DetailData, error) {
	d := model.NewDetailData()

	partitions, err := disk.Partitions(false)
	if err != nil {
		logger.Errorf("get partitions error %+v", err)
	} else {
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
			d.Values["disk_total"] = diskTotal
			d.Values["disk_used"] = diskUsed
			d.Values["disk_util"] = 100 * float64(diskUsed) / float64(diskTotal)
		} else {
			d.Values["disk_total"] = 0
			d.Values["disk_used"] = 0
			d.Values["disk_util"] = 0
		}
	}

	//stats, err := disk.IOCounters()
	//if err == nil {
	//	newState.stats = stats
	//	if state != nil && state.stats != nil {
	//		fmt.Println(stats)
	//		for dev, stat2 := range stats {
	//			if stat1, ok := state.stats[dev]; ok {
	//				fmt.Println("readCount", dev, stat2.ReadCount-stat1.ReadCount)
	//				fmt.Println("readBytes", dev, stat2.ReadBytes-stat1.ReadBytes)
	//				fmt.Println("writeCount", dev, stat2.WriteCount-stat1.WriteCount)
	//				fmt.Println("writeBytes", dev, stat2.WriteBytes-stat1.WriteBytes)
	//			}
	//		}
	//	}
	//}

	return model.MakeDetailDataSlice(d), nil
}
