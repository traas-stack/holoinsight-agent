/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFrequency(t *testing.T) {
	ilf := newIntelligentFrequency(3, 5)

	ilf.set(false)
	assert.False(t, ilf.isDown())

	ilf.set(false)
	assert.False(t, ilf.isDown())

	ilf.set(false)
	assert.True(t, ilf.isDown())

	ilf.set(false)
	assert.True(t, ilf.isDown())

	ilf.set(false)
	assert.False(t, ilf.isDown())

	ilf.set(false)
	assert.True(t, ilf.isDown())
}
