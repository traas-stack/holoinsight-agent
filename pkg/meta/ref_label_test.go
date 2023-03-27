package meta

import (
	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"testing"
)

func TestRefLabel(t *testing.T) {
	labels := map[string]string{}
	to := map[string]string{}
	item := &appconfig.DataConfig_Metric_RefLabels_Item{
		Key:          "d",
		Labels:       []string{"a", "b"},
		DefaultValue: "c",
	}

	RefLabel(item, labels, to)
	assert.Equal(t, "c", to["d"])

	delete(to, "d")
	labels["b"] = "2"
	RefLabel(item, labels, to)
	assert.Equal(t, "2", to["d"])

	delete(to, "d")
	labels["a"] = "3"
	RefLabel(item, labels, to)
	assert.Equal(t, "3", to["d"])
}
