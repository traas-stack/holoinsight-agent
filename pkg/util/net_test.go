/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReplaceHost(t *testing.T) {
	assert.Equal(t, "2.2.2.2:2222", ReplaceHost("1.1.1.1:2222", "2.2.2.2"))
}
