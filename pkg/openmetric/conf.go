/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package openmetric

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"net/url"
	"strconv"
)

type (
	openMetricConf struct {
		// 这里有一个 typo, 应该是 scheme 才对, 于是我们做下兼容
		Schema          string            `json:"schema"`
		Scheme          string            `json:"scheme"`
		MetricsPath     string            `json:"metricsPath"`
		ScrapeInterval  string            `json:"scrapeInterval"`
		ScrapeTimeout   string            `json:"scrapeTimeout"`
		ScrapePort      string            `json:"scrapePort"`
		Targets         []string          `json:"targets"`
		Labels          map[string]string `json:"labels"`
		HonorLabels     bool              `json:"honorLabels"`
		HonorTimestamps bool              `json:"honorTimestamps"`
		Params          url.Values        `json:"params"`
	}
)

func convertToScrapeConfig(t *collecttask.CollectTask) (*config.ScrapeConfig, error) {
	omc := &openMetricConf{}
	err := json.Unmarshal(t.Config.Content, omc)
	if err != nil {
		return nil, err
	}

	// 服务端数据不够规范, 进行一些修正
	if _, err := strconv.ParseInt(omc.ScrapeInterval, 10, 64); err == nil {
		omc.ScrapeInterval += "s"
	}
	if _, err := strconv.ParseInt(omc.ScrapeTimeout, 10, 64); err == nil {
		omc.ScrapeTimeout += "s"
	}
	scheme := omc.Schema
	if scheme == "" {
		scheme = omc.Scheme
	}
	if scheme == "" {
		scheme = "http"
	}
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("invalid schema %s", scheme)
	}

	scrapeInterval, err := model.ParseDuration(omc.ScrapeInterval)
	if err != nil {
		return nil, err
	}

	scrapeTimeout, err := model.ParseDuration(omc.ScrapeTimeout)
	if err != nil {
		return nil, err
	}

	var targets []model.LabelSet

	targets = append(targets, model.LabelSet{
		model.AddressLabel: model.LabelValue(fmt.Sprintf("%s:%s", t.Target.GetIP(), omc.ScrapePort)),
	})

	sc := &config.ScrapeConfig{
		JobName:         t.Key,
		HonorLabels:     omc.HonorLabels,
		HonorTimestamps: omc.HonorTimestamps,
		Params:          omc.Params,
		ScrapeInterval:  scrapeInterval,
		ScrapeTimeout:   scrapeTimeout,
		MetricsPath:     omc.MetricsPath,
		Scheme:          scheme,
		ServiceDiscoveryConfigs: discovery.Configs{
			discovery.StaticConfig{
				{
					Targets: targets,
				},
			},
		},
	}

	addNetProxy(sc)
	return sc, nil
}
