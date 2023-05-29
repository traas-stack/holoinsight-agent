/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package base

import "github.com/traas-stack/holoinsight-agent/pkg/collectconfig"

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
		MetricBlacklist []string `json:"metricBlacklist,omitempty"`
	}
)
