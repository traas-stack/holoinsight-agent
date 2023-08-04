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
	"github.com/prometheus/prometheus/pkg/relabel"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"net/url"
	"strconv"
)

type (
	openMetricConf struct {
		Schema               string            `json:"schema"`
		Scheme               string            `json:"scheme"`
		MetricsPath          string            `json:"metricsPath"`
		ScrapeInterval       string            `json:"scrapeInterval"`
		ScrapeTimeout        string            `json:"scrapeTimeout"`
		ScrapePort           string            `json:"scrapePort"`
		Targets              []string          `json:"targets"`
		Labels               map[string]string `json:"labels"`
		HonorLabels          bool              `json:"honorLabels"`
		HonorTimestamps      bool              `json:"honorTimestamps"`
		Params               url.Values        `json:"params"`
		RelabelConfigs       []*relabelConfig  `json:"relabelConfigs"`
		MetricRelabelConfigs []*relabelConfig  `json:"metricRelabelConfigs"`
	}
	relabelConfig struct {
		SourceLabels []string `json:"sourceLabels,omitempty"`
		Separator    string   `json:"separator,omitempty"`
		Regex        string   `json:"regex,omitempty"`

		// Modulus uint64

		TargetLabel string `json:"targetLabel,omitempty"`
		Replacement string `json:"replacement,omitempty"`
		Action      string `json:"action,omitempty"`
	}
)

func convertToScrapeConfig(t *collecttask.CollectTask) (*config.ScrapeConfig, error) {
	omc := &openMetricConf{}
	err := json.Unmarshal(t.Config.Content, omc)
	if err != nil {
		return nil, err
	}

	if _, err := strconv.ParseInt(omc.ScrapeInterval, 10, 64); err == nil {
		omc.ScrapeInterval += "s"
	}
	if _, err := strconv.ParseInt(omc.ScrapeTimeout, 10, 64); err == nil {
		omc.ScrapeTimeout += "s"
	}
	// fix the typo
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

	var discoveryConfig discovery.Config
	if t.Target.IsTypePod() {
		discoveryConfig = &onePodDiscoveryConfig{
			namespace: t.Target.GetNamespace(),
			name:      t.Target.GetPodName(),
			port:      omc.ScrapePort,
		}
	} else {
		discoveryConfig = discovery.StaticConfig{
			{
				Targets: []model.LabelSet{{model.AddressLabel: model.LabelValue(fmt.Sprintf("%s:%s", t.Target.GetIP(), omc.ScrapePort))}},
			},
		}
	}

	sc := &config.ScrapeConfig{
		JobName:                 t.Key,
		HonorLabels:             omc.HonorLabels,
		HonorTimestamps:         omc.HonorTimestamps,
		Params:                  omc.Params,
		ScrapeInterval:          scrapeInterval,
		ScrapeTimeout:           scrapeTimeout,
		MetricsPath:             omc.MetricsPath,
		Scheme:                  scheme,
		ServiceDiscoveryConfigs: discovery.Configs{discoveryConfig},
	}

	for _, rc := range omc.RelabelConfigs {
		c, err := convertRelabelConfig(rc)
		if err != nil {
			return nil, err
		}
		sc.RelabelConfigs = append(sc.RelabelConfigs, c)
	}

	for _, rc := range omc.MetricRelabelConfigs {
		c, err := convertRelabelConfig(rc)
		if err != nil {
			return nil, err
		}
		sc.MetricRelabelConfigs = append(sc.MetricRelabelConfigs, c)
	}

	addNetProxy(sc)
	return sc, nil
}

func convertRelabelConfig(rc *relabelConfig) (*relabel.Config, error) {
	if rc.Action == "" {
		rc.Action = string(relabel.Replace)
	}
	if rc.Separator == "" {
		rc.Separator = ";"
	}
	if rc.Regex == "" {
		rc.Regex = "(.*)"
	}
	if rc.Replacement == "" {
		rc.Replacement = "$1"
	}
	var sourceLabels model.LabelNames
	for _, label := range rc.SourceLabels {
		sourceLabels = append(sourceLabels, model.LabelName(label))
	}

	regex, err := relabel.NewRegexp(rc.Regex)
	if err != nil {
		return nil, err
	}

	return &relabel.Config{
		SourceLabels: sourceLabels,
		TargetLabel:  rc.TargetLabel,
		Action:       relabel.Action(rc.Action),
		Separator:    rc.Separator,
		Replacement:  rc.Replacement,
		Regex:        regex,
	}, nil
}
