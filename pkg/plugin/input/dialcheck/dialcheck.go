/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package dialcheck

import (
	"encoding/json"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"net"
	"time"
)

type (
	Config struct {
		Network     string        `json:"network"`
		Addr        string        `json:"addr"`
		Timeout     time.Duration `json:"timeout"`
		Times       int           `json:"times"`
		NetworkMode string        `json:"networkMode"`
	}

	Input struct {
		Config *Config
	}
)

const (
	defaultTimeout             = 3 * time.Second
	defaultTimes               = 1
	maxTimes                   = 5
	HelperInputProxyConfigType = "dialcheck"
)

func (i *Input) NetworkMode() string {
	return i.Config.NetworkMode
}

func (i *Input) SerializeRequest() (interface{}, string, []byte, time.Duration, error) {
	configBytes, err := json.Marshal(i.Config)
	return nil, HelperInputProxyConfigType, configBytes, i.getTimeout(), err
}

func (i *Input) getTimeout() time.Duration {
	timeout := defaultTimeout
	if i.Config.Timeout > 0 {
		timeout = i.Config.Timeout
	}
	return timeout
}

func (i *Input) ExecuteRequest(bytes []byte) ([]byte, error) {
	config := &Config{}
	err := json.Unmarshal(bytes, config)
	if err != nil {
		return nil, err
	}
	i.Config = config

	ma := api.NewMemoryAccumulator()
	err = i.Collect(ma)
	if err != nil {
		return nil, err
	}

	return json.Marshal(ma.Memory)
}

func (i *Input) ProcessResponse(_ interface{}, respBytes []byte, err error, accumulator api.Accumulator) error {
	if err != nil {
		return err
	}
	return api.NsEnterHelpProcesResponse(respBytes, accumulator)
}

func (i *Input) Collect(a api.Accumulator) error {
	timeout := i.getTimeout()

	times := defaultTimes
	if i.Config.Times > 0 && i.Config.Times < maxTimes {
		times = i.Config.Times
	}

	up := 0
	anyUp := 0
	totalCost := time.Duration(0)

	for j := 0; j < times; j++ {

		begin := time.Now()
		conn, err := net.DialTimeout(i.Config.Network, i.Config.Addr, timeout)
		cost := time.Now().Sub(begin)
		totalCost += cost

		if err == nil {
			conn.Close()
			up++
			anyUp = 1
		}
	}

	a.AddMetric(&model.Metric{
		Name:  "dialcheck_up",
		Tags:  map[string]string{},
		Value: float64(anyUp),
	})
	a.AddMetric(&model.Metric{
		Name:  "dialcheck_down",
		Tags:  map[string]string{},
		Value: float64(1 - anyUp),
	})
	a.AddMetric(&model.Metric{
		Name:  "dialcheck_cost",
		Tags:  map[string]string{},
		Value: float64(int(totalCost.Milliseconds()) / times),
	})
	return nil
}
