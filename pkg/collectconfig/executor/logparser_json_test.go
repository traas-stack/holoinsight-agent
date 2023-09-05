/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJson(t *testing.T) {
	m := map[string]interface{}{"a": 1}
	line := `sdf`
	err := json.Unmarshal([]byte(line), &m)
	assert.Error(t, err)
	assert.Len(t, m, 1)
	assert.Equal(t, 1, m["a"])

	line = `{"a":2,"b":3.a}`
	err = json.Unmarshal([]byte(line), &m)
	assert.Error(t, err)
	assert.Len(t, m, 1)
	assert.Equal(t, 1, m["a"])

	line = `{"b":2}`
	err = json.Unmarshal([]byte(line), &m)
	assert.NoError(t, err)
	assert.Len(t, m, 2)
	assert.Equal(t, 1, m["a"])
	assert.Equal(t, 2.0, m["b"])

	line = `{"a":2,"b":3}`
	err = json.Unmarshal([]byte(line), &m)
	assert.NoError(t, err)
	assert.Len(t, m, 2)
	assert.Equal(t, 2.0, m["a"])
	assert.Equal(t, 3.0, m["b"])
}
