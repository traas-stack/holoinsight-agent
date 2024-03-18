/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package base

import "github.com/traas-stack/holoinsight-agent/pkg/collectconfig"

const (
	ValueManipulationDelta = "DELTA"
	ValueManipulationRate  = "RATE"
)

type (
	Conf struct {
		Name        string                                 `json:"name,omitempty"`
		Type        string                                 `json:"type,omitempty"`
		ExecuteRule collectconfig.ExecuteRule              `json:"executeRule,omitempty"`
		RefMetas    map[string]*collectconfig.ElectRegMeta `json:"refMetas,omitempty"`
		Transform   Transform                              `json:"transform,omitempty"`
	}

	Transform struct {
		// MetricPrefix adds a prefix to the original metric name
		MetricPrefix string `json:"metricPrefix,omitempty"`
		// MetricFormat is the format of metric name.
		// Such as 'system_%s'
		MetricFormat string `json:"metricFormat,omitempty"`
		// Metric white list
		MetricWhitelist []string `json:"metricWhitelist,omitempty"`
		// Metric black list
		MetricBlacklist []string                 `json:"metricBlacklist,omitempty"`
		MetricConfigs   map[string]*MetricConfig `json:"metricConfigs,omitempty"`
		Scripts         []string                 `json:"scripts"`
	}
	MetricConfig struct {
		// DELTA = currentValue - lastValue
		// RATE = DELTA / SECONDS
		// GAUGE: the collected value is an instantaneous value
		ValueManipulation string `json:"valueManipulation"`
		// If KeepNegative is false, fix the result to 0 when the result is negative
		KeepNegative bool `json:"keepNegative"`
	}
)
