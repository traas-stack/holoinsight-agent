/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"testing"
)

func TestElectRefNameJson(t *testing.T) {
	e, err := parseElect0(&collectconfig.Elect{
		Type: "refName",
		RefName: &collectconfig.RefName{
			Name: "$.username",
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, e)

	e.Init()

	ctx := &LogContext{
		columnMap: make(map[string]interface{}),
	}

	assert.NoError(t, json.Unmarshal([]byte(`{"username":"xzchaoo"}`), &ctx.columnMap))
	x, err := e.Elect(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "xzchaoo", x)

	ctx.columnMap = make(map[string]interface{})
	assert.NoError(t, json.Unmarshal([]byte(`{"username2":"xzchaoo"}`), &ctx.columnMap))
	x, err = e.Elect(ctx)
	assert.Error(t, err)
	assert.Nil(t, x)

	x, err = e.ElectString(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "", x)
}
