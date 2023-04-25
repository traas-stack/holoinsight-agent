/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type (
	FilePathFormatter struct {
		r *regexp.Regexp
	}
)

var (
	stdFilePathFormatter = newFilePathFormatter()
)

func newFilePathFormatter() *FilePathFormatter {
	r := regexp.MustCompile("{[a-zA-Z:]+?}")
	return &FilePathFormatter{
		r: r,
	}
}

func (f *FilePathFormatter) ContainsFormat(s string) bool {
	return f.r.MatchString(s)
}

func (f *FilePathFormatter) Format(s string) string {
	now := time.Now()
	return f.r.ReplaceAllStringFunc(s, func(s string) string {
		switch s {
		case "{time:yyyy}":
			return strconv.FormatInt(int64(now.Year()), 10)
		case "{time:MM}":
			return fmt.Sprintf("%02d", now.Month())
		case "{time:M}":
			return strconv.FormatInt(int64(now.Month()), 10)
		case "{time:dd}":
			return fmt.Sprintf("%02d", now.Day())
		case "{time:d}":
			return strconv.FormatInt(int64(now.Day()), 10)
		case "{time:HH}":
			return fmt.Sprintf("%02d", now.Hour())
		case "{time:H}":
			return strconv.FormatInt(int64(now.Hour()), 10)
		default:
			return s
		}
	})
}
