/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package dialcheck

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"net"
	"sync"
	"time"
)

type (
	Config struct {
		Network     string        `json:"network"`
		Host        string        `json:"host"`
		Ports       []int         `json:"ports"`
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

func (i *Input) GetDefaultPrefix() string {
	return "dialcheck_"
}

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

	return json.Marshal(ma.Metrics)
}

func (i *Input) ProcessResponse(_ interface{}, respBytes []byte, err error, accumulator api.Accumulator) error {
	if err != nil {
		return err
	}
	return api.NsEnterHelpProcesResponse(respBytes, accumulator)
}

func (i *Input) Collect(a api.Accumulator) error {
	var wg sync.WaitGroup
	subAccumulators := make([]*api.MemoryAccumulator, len(i.Config.Ports))
	for index, port := range i.Config.Ports {
		ma := api.NewMemoryAccumulator()
		subAccumulators[index] = ma
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			i.collectOnePort(port, ma)
		}(port)
	}
	wg.Wait()
	for _, ma := range subAccumulators {
		for _, metric := range ma.Metrics {
			a.AddMetric(metric)
		}

	}
	return nil
}

func (i *Input) collectOnePort(port int, a api.Accumulator) {
	timeout := i.getTimeout()

	times := defaultTimes
	if i.Config.Times > 0 && i.Config.Times < maxTimes {
		times = i.Config.Times
	}

	up := 0
	anyUp := 0
	totalCost := time.Duration(0)

	addr := fmt.Sprintf("%s:%d", i.Config.Host, port)
	for j := 0; j < times; j++ {

		begin := time.Now()
		conn, err := net.DialTimeout(i.Config.Network, addr, timeout)
		cost := time.Now().Sub(begin)
		totalCost += cost

		if err == nil {
			conn.Close()
			up++
			anyUp = 1
		}
	}

	portStr := cast.ToString(port)

	a.AddMetric(&model.Metric{
		Name: "up",
		Tags: map[string]string{
			"port": portStr,
		},
		Value: float64(anyUp),
	})
	a.AddMetric(&model.Metric{
		Name: "down",
		Tags: map[string]string{
			"port": portStr,
		},
		Value: float64(1 - anyUp),
	})
	a.AddMetric(&model.Metric{
		Name: "cost",
		Tags: map[string]string{
			"port": portStr,
		},
		Value: float64(int(totalCost.Milliseconds()) / times),
	})
}

func (i *Input) GenerateErrorMetrics(a api.Accumulator) {
	for _, port := range i.Config.Ports {
		portStr := cast.ToString(port)
		a.AddMetric(&model.Metric{
			Name: "up",
			Tags: map[string]string{
				"port": portStr,
			},
			Value: 0,
		})
		a.AddMetric(&model.Metric{
			Name: "down",
			Tags: map[string]string{
				"port": portStr,
			},
			Value: 1,
		})
	}
}
