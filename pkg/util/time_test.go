/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	d, err := ParseDuration("100ms")
	assert.Nil(t, err)
	assert.Equal(t, 100*time.Millisecond, d)

	d, err = ParseDuration("100")
	assert.Nil(t, err)
	assert.Equal(t, 100*time.Millisecond, d)

	d, err = ParseDuration(100)
	assert.Nil(t, err)
	assert.Equal(t, 100*time.Millisecond, d)

	d, err = ParseDuration("100.0")
	assert.Nil(t, err)
	assert.Equal(t, 100*time.Millisecond, d)

	d, err = ParseDuration("-100.0")
	assert.NotNil(t, err)
}
