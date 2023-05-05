/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRefVar(t *testing.T) {
	e := &xRefVar{name: "a"}

	s, err := e.ElectString(&LogContext{})
	assert.NoError(t, err)
	assert.Equal(t, "", s)

	s, err = e.ElectString(&LogContext{vars: map[string]interface{}{"a": "holoinsight"}})
	assert.NoError(t, err)
	assert.Equal(t, "holoinsight", s)

	s, err = e.ElectString(&LogContext{vars: map[string]interface{}{"b": "holoinsight"}})
	assert.NoError(t, err)
	assert.Equal(t, "", s)

}
