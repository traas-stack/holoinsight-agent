package meta

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
)

func RefLabels(items []*appconfig.DataConfig_Metric_RefLabels_Item, labels map[string]string, to map[string]string) {
	for _, item := range items {
		RefLabel(item, labels, to)
	}
}

func RefLabel(item *appconfig.DataConfig_Metric_RefLabels_Item, labels map[string]string, to map[string]string) bool {
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
