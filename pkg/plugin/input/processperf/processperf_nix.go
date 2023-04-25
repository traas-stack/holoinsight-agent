/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

//go:build linux || darwin
// +build linux darwin

package processperf

import (
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/shirou/gopsutil/v3/process"
	"syscall"
)

func fillConnectionsMetrics(result []*model.DetailData, p *process.Process, baseTags map[string]string) []*model.DetailData {
	// TODO 这段代码linux特定
	if conns, err := p.Connections(); err == nil {
		streamCount := map[string]int{}
		dgramCount := map[string]int{}
		for i := range conns {
			c := &conns[i]
			if c.Type == syscall.SOCK_STREAM {
				k := c.Status
				streamCount[k] = streamCount[k] + 1
			} else if c.Type == syscall.SOCK_DGRAM {
				k := c.Status
				dgramCount[k] = dgramCount[k] + 1
			} else {
				continue
			}
		}
		for countType, count := range map[string]map[string]int{
			"SOCK_STREAM": streamCount,
			"SOCK_DGRAM":  dgramCount,
		} {
			for s, i := range count {
				tags := util.CopyStringMapCap(baseTags, len(baseTags)+2)
				tags["type"] = countType
				tags["status"] = s
				dd2 := &model.DetailData{
					// TODO time?
					Timestamp: 0,
					Tags:      tags,
					Values: map[string]interface{}{
						"connections": i,
					},
				}
				result = append(result, dd2)
			}
		}
	}
	return result
}
