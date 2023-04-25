/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"fmt"
	"testing"
)

func TestRegexpFileMatcher(t *testing.T) {
	m, err := NewRegexpFileMatcher("/Users/xzchaoo/logs", "^/Users/xzchaoo/logs/(?P<type>[^/]+)/(?P<basename>[^/]+).log$", -1, 10)
	if err != nil {
		panic(err)
	}
	fmt.Println(m.Find())
}
