/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestFormat2FileMatcher(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "TestFormat2FileMatcher-*")
	if err != nil {
		panic(err)
	}
	fmt.Println(tempdir)
	os.MkdirAll(filepath.Join(tempdir, "type1"), 0755)
	os.MkdirAll(filepath.Join(tempdir, "type2"), 0755)
	os.Create(filepath.Join(tempdir, "type1", "foo.log"))
	os.Create(filepath.Join(tempdir, "type1", "bar.log"))
	os.Create(filepath.Join(tempdir, "type2", "bar.log"))

	m, err := NewFormat2FileMatcher(tempdir, tempdir+"/{type}/{name}.log")
	if err != nil {
		panic(err)
	}

	result, _, err := m.Find()
	if err != nil {
		panic(err)
	}
	assert.Equal(t, 3, len(result))
}
