/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	cc "github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"testing"
)

func TestVars_parse(t *testing.T) {
	conf, err := loadTransformConf("transforms/mapping1.yaml")
	assert.NoError(t, err)
	varsProcessor, err := parseVars(&cc.Vars{
		Vars: []*cc.Var{
			{
				Name: "v1",
				Elect: &cc.Elect{
					Type:      "context",
					Transform: conf,
				},
			},
		},
	})
	assert.NoError(t, err)

	vars, err := varsProcessor.process(&LogContext{contextValue: "a"})
	assert.NoError(t, err)
	fmt.Println(vars)
	assert.Equal(t, "aa", vars["v1"])

	vars, err = varsProcessor.process(&LogContext{contextValue: "b"})
	assert.NoError(t, err)
	fmt.Println(vars)
	assert.Equal(t, "bb", vars["v1"])

	vars, err = varsProcessor.process(&LogContext{contextValue: "c"})
	assert.NoError(t, err)
	fmt.Println(vars)
	assert.Equal(t, "xx", vars["v1"])
}
