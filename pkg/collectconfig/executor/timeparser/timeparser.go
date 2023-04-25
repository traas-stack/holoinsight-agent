/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package timeparser

import (
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"time"
)

type (
	TimeParser struct {
	}
)

// 解析时间
func ParseTime(line string) int64 {
	layout := "2006-01-02 15:04:05"
	t, err := time.ParseInLocation(layout, line[:len(layout)], time.Local)
	if err != nil {
		return -1
	} else {
		return util.ToMS(t)
	}
}
