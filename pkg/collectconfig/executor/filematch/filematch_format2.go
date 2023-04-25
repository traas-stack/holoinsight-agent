/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"errors"
	"fmt"
	"regexp"
)

type (
	Format2FileMatcher struct {
		basedir string
		pattern string
		r       *regexp.Regexp
	}
)

var (
	format2Regexp = regexp.MustCompile("{[a-zA-Z0-9]+}")
)

func NewFormat2FileMatcher(basedir string, pattern string) (FileMatcher, error) {
	if basedir == "" || basedir == "/" {
		return nil, errors.New("basedir must not be root")
	}

	regexpPattern := format2Regexp.ReplaceAllStringFunc(pattern, func(s string) string {
		key := s[1 : len(s)-1]
		return fmt.Sprintf("(?P<%s>[^/]+)", key)
	})

	return NewRegexpFileMatcher(basedir, regexpPattern, -1, -1)
}
