/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"testing"
)

func TestMappingV1(t *testing.T) {
	filter, err := parseTransformFilter(&collectconfig.TransformFilterConf{
		MappingV1: &collectconfig.TransformFilterMappingV1{
			Mappings: map[string]string{
				"a": "aa",
			},
			DefaultValue: "xxx",
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, filter)

	v, err := filter.Filter(&LogContext{contextValue: "a"})
	assert.NoError(t, err)
	assert.Equal(t, "aa", v)

	v, err = filter.Filter(&LogContext{contextValue: "b"})
	assert.NoError(t, err)
	assert.Equal(t, "xxx", v)
}
