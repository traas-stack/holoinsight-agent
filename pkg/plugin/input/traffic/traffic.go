package traffic

import (
	"bytes"
	"encoding/gob"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

type (
	trafficInput struct {
		input.BaseInput
		state *trafficState
	}
	trafficState struct {
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

func (i *trafficInput) SerializeState() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := gob.NewEncoder(buf).Encode(i.state)
	return buf.Bytes(), err
}

func (i *trafficInput) DeserializeState(in []byte) error {
	i.state = &trafficState{}
	return gob.NewDecoder(bytes.NewBuffer(in)).Decode(i.state)
}

func (i *trafficInput) Collect(ctx *input.CollectContext) ([]*model.DetailData, error) {
	now := util.CurrentMS()

	state := i.state
	newState := &trafficState{
		Time: now,
	}
	i.state = newState
	mills := int64(0)
	if state != nil {
		mills = now - state.Time
	}

	d := model.NewDetailData()

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
			d.Values["traffic_bytin"] = (st.BytesRecv - state.Net.BytesRecv) * 1000 / uint64(mills)
			d.Values["traffic_bytout"] = (st.BytesSent - state.Net.BytesSent) * 1000 / uint64(mills)

			// pkg/s
			d.Values["traffic_pktin"] = (st.PacketsRecv - state.Net.PacketsRecv) * 1000 / uint64(mills)
			d.Values["traffic_pktout"] = (st.PacketsSent - state.Net.PacketsSent) * 1000 / uint64(mills)
			d.Values["traffic_pktout"] = (st.PacketsSent - state.Net.PacketsSent) * 1000 / uint64(mills)
			d.Values["traffic_pktdrp"] = (st.Drop - state.Net.Drop) * 1000 / uint64(mills)
			d.Values["traffic_pkterr"] = (st.Err - state.Net.Err) * 1000 / uint64(mills)
		}
	}

	return model.MakeDetailDataSlice(d), nil
}
