/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package meta

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
)

// RefLabels ref labels as tags
func RefLabels(items []*appconfig.DataConfig_Metric_RefLabels_Item, labels map[string]string, to map[string]string) {
	for _, item := range items {
		RefLabel(item, labels, to)
	}
}

// RefLabel ref labels as tags
func RefLabel(item *appconfig.DataConfig_Metric_RefLabels_Item, labels map[string]string, to map[string]string) bool {
	if _, exist := to[item.Key]; exist {
		return false
	}

	var labelValue string
	for _, label := range item.Labels {
		labelValue = labels[label]
		if labelValue != "" {
			break
		}
	}
	if labelValue == "" {
		labelValue = item.DefaultValue
	}

	if labelValue != "" {
		to[item.Key] = labelValue
		return true
	}
	return false
}
