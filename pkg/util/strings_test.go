/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ConvertStringSliceToHashSet(t *testing.T) {
	assert.Nil(t, ConvertStringSliceToHashSet(nil))

	x := ConvertStringSliceToHashSet([]string{})
	assert.NotNil(t, x)
	assert.Equal(t, 0, len(x))

	x = ConvertStringSliceToHashSet([]string{"a"})
	assert.Equal(t, 1, len(x))

	_, ok := x["a"]
	assert.True(t, ok)
}
