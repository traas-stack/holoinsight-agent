/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package tcp

import (
	"github.com/shirou/gopsutil/v3/net"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

type (
	tcpInput struct {
		state *internalState
	}
	internalState struct {
		Time int64
		Tcp  map[string]int64
		Udp  map[string]int64
	}
)

func (i *tcpInput) GetDefaultPrefix() string {
	return ""
}

func (i *tcpInput) Collect(a api.Accumulator) error {
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
	protoCounters, err := net.ProtoCounters([]string{"tcp", "udp"})
	if err != nil {
		//if err != errNotImplementedError {
		//	// mac上不支持 要冷却一下错误信息
		//	// 2022/03/07 16:16:12.453732 get net ProtoCounters error not implemented yet
		//}
		if !util.IsNotImplemented(err) {
			logger.Errorf("get net ProtoCounters error %+v", err)
		}
		// TODO 是否补零
	} else {
		for i := range protoCounters {
			c := &protoCounters[i]
			switch c.Protocol {
			case "tcp":
				//{"protocol":"tcp","stats":{"ActiveOpens":63673644,"AttemptFails":13740544,"CurrEstab":32,"EstabResets":1871859,"InCsumErrors":528,"InErrs":16144,"InSegs":19572731894,"MaxConn":-1,"OutRsts":29750796,"OutSegs":46807245439,"PassiveOpens":18462548,"RetransSegs":35597012,"RtoAlgorithm":1,"RtoMax":120000,"RtoMin":200}}
				stats := c.Stats
				stats["totalSegs"] = stats["InSegs"] + stats["OutSegs"]
				newState.Tcp = stats

				// 这几个是状态值 作差无意义
				values["tcp_CurrEs"] = c.Stats["CurrEstab"]

				if state != nil && state.Tcp != nil {
					values["tcp_active"] = (stats["ActiveOpens"] - state.Tcp["ActiveOpens"]) * 1000 / mills
					values["tcp_pasive"] = (stats["PassiveOpens"] - state.Tcp["PassiveOpens"]) * 1000 / mills
					values["tcp_EstRes"] = (stats["EstabResets"] - state.Tcp["EstabResets"]) * 1000 / mills
					values["tcp_AtmpFa"] = (stats["AttemptFails"] - state.Tcp["AttemptFails"]) * 1000 / mills
					values["tcp_iseg"] = (stats["InSegs"] - state.Tcp["InSegs"]) * 1000 / mills
					values["tcp_oseg"] = (stats["OutSegs"] - state.Tcp["OutSegs"]) * 1000 / mills

					if stats["totalSegs"] > state.Tcp["totalSegs"] {
						values["tcp_retran"] = 100 * float64(stats["RetransSegs"]-state.Tcp["RetransSegs"]) / float64(stats["totalSegs"]-state.Tcp["totalSegs"])
					} else {
						values["tcp_retran"] = 0.0
					}
				} else {
					values["tcp_EstRes"] = 0.0
					values["tcp_AtmpFa"] = 0.0
					values["tcp_retran"] = 0.0
					values["tcp_iseg"] = 0.0
					// d.Values["tcp."] = 0.0
				}
			case "udp":
				// {"protocol":"udp","stats":{"InCsumErrors":0,"InDatagrams":13160981,"InErrors":0,"NoPorts":24922,"OutDatagrams":39821059,"RcvbufErrors":0,"SndbufErrors":0}}
				stats := c.Stats
				newState.Udp = stats
				if state != nil && state.Udp != nil {
					values["udp_idgm"] = (stats["InDatagrams"] - state.Udp["InDatagrams"]) * 1000 / mills
					values["udp_odgm"] = (stats["OutDatagrams"] - state.Udp["OutDatagrams"]) * 1000 / mills
				}
			}
		}
	}

	input.AddMetrics(a, values)
	return nil
}
