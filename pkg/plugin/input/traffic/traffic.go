/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package traffic

import (
	"github.com/shirou/gopsutil/v3/net"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

type (
	trafficInput struct {
		state *internalState
	}
	internalState struct {
		Time int64
		Net  *netTimesStat
	}
	netTimesStat struct {
		BytesSent   uint64
		BytesRecv   uint64
		PacketsSent uint64
		PacketsRecv uint64
		Drop        uint64
		Err         uint64
	}
)

func (i *trafficInput) GetDefaultPrefix() string {
	return ""
}

func (i *trafficInput) Collect(a api.Accumulator) error {
	now := util.CurrentMS()

	state := i.state
	newState := &internalState{
		Time: now,
	}
	i.state = newState
	mills := int64(0)
	if state != nil {
		mills = now - state.Time
	}

	values := make(map[string]interface{})

	ioCounters, err := net.IOCounters(true)
	if err != nil {
		logger.Errorf("get net stat error %+v", err)
	} else {
		st := netTimesStat{}
		for i := range ioCounters {
			stat := &ioCounters[i]
			st.BytesSent += stat.BytesSent
			st.BytesRecv += stat.BytesRecv
			st.PacketsSent += stat.PacketsSent
			st.PacketsRecv += stat.PacketsRecv
			st.Drop += stat.Dropin + stat.Dropout
			st.Err += stat.Errin + stat.Errout
		}
		newState.Net = &st

		if state != nil && state.Net != nil {
			// 这里还需要除以时间才是 bytes/s
			values["traffic_bytin"] = (st.BytesRecv - state.Net.BytesRecv) * 1000 / uint64(mills)
			values["traffic_bytout"] = (st.BytesSent - state.Net.BytesSent) * 1000 / uint64(mills)

			// pkg/s
			values["traffic_pktin"] = (st.PacketsRecv - state.Net.PacketsRecv) * 1000 / uint64(mills)
			values["traffic_pktout"] = (st.PacketsSent - state.Net.PacketsSent) * 1000 / uint64(mills)
			values["traffic_pktout"] = (st.PacketsSent - state.Net.PacketsSent) * 1000 / uint64(mills)
			values["traffic_pktdrp"] = (st.Drop - state.Net.Drop) * 1000 / uint64(mills)
			values["traffic_pkterr"] = (st.Err - state.Net.Err) * 1000 / uint64(mills)
		}
	}

	input.AddMetrics(a, values)
	return nil
}
