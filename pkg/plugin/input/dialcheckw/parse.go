/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package dialcheckw

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/telegraf/providers"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/dialcheck"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/inputproxy"
	"time"
)

const (
	defaultTimeout = 3 * time.Second
	defaultTimes   = 1
	maxTimes       = 5
)

type (
	Config struct {
		Network     string `json:"network"`
		Port        int    `json:"port"`
		Timeout     int    `json:"timeout"`
		Times       int    `json:"times"`
		NetworkMode string `json:"networkMode"`
	}
)

func init() {
	providers.Register("dialcheck", Parse)
	inputproxy.Register("dialcheck", func() api.InputExtNsEnter {
		return &dialcheck.Input{}
	})
}

func Parse(task *collecttask.CollectTask) (interface{}, error) {
	config := &Config{}
	err := json.Unmarshal(task.Config.Content, config)
	if err != nil {
		return nil, err
	}
	network := config.Network
	if network == "" {
		return nil, errors.New("network is empty")
	}

	timeout := defaultTimeout
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Millisecond
	}

	times := defaultTimes
	if config.Times > 0 {
		times = config.Times
	}
	if times > maxTimes {
		times = maxTimes
	}

	target := task.Target
	var addr string
	port := config.Port
	if target.IsTypePod() {
		addr = fmt.Sprintf("%s:%d", target.GetIP(), port)
	} else if target.IsTypeLocalhost() {
		addr = fmt.Sprintf("localhost:%d", port)
	} else {
		return nil, fmt.Errorf("unsupported target type %v", target)
	}

	return &dialcheck.Input{Config: &dialcheck.Config{
		Network:     network,
		Addr:        addr,
		Timeout:     timeout,
		Times:       times,
		NetworkMode: config.NetworkMode,
	}}, nil
}
