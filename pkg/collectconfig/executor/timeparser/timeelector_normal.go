/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package timeparser

import (
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"strings"
	"time"
)

type (
	SimpleTimeElector struct {
		YearOffsetAdjust int
		layouts          []string
	}
)

func (n *SimpleTimeElector) ParseTimeStyle(line string) *TimeStyle {
	for _, layout := range n.layouts {
		ts := n.parseTimeStyle0(layout, line)
		if ts != nil {
			ts.Layout = layout
			return ts
		}
	}
	return nil
}

func (n *SimpleTimeElector) parseTimeStyle0(layout string, line string) *TimeStyle {
	if len(line) < len(layout) {
		return nil
	}
	start := findTimeStart(line)
	if start < n.YearOffsetAdjust {
		for i := 0; i < 8; i++ {
			_, err := time.ParseInLocation(layout, line[i:i+len(layout)], time.Local)
			if err == nil {
				ts := &TimeStyle{}
				fillYearLeft(ts, line, i+n.YearOffsetAdjust)
				return ts
			}
		}
		return nil
	}
	start -= n.YearOffsetAdjust

	end := start + len(layout)
	if end > len(line) {
		return nil
	}

	timeStr := line[start:end]

	_, err := time.ParseInLocation(layout, timeStr, time.Local)
	if err != nil {
		return nil
	}
	ts := &TimeStyle{}
	fillYearLeft(ts, line, start+n.YearOffsetAdjust)
	return ts
}

func (n *SimpleTimeElector) Parse(style *TimeStyle, line string, tz *time.Location) int64 {
	if len(line) < len(style.Layout) {
		return ParseError
	}

	if style.YearLeftIndex == lineStart {
		t, err := time.ParseInLocation(style.Layout, line[:len(style.Layout)], tz)
		if err != nil {
			return ParseError
		}
		return util.ToMS(t)
	}

	start := 0
	for i := 0; i <= style.YearLeftIndex; i++ {
		index := strings.Index(line[start:], style.YearLeft)
		if index < 0 {
			return ParseError
		}
		start += index + len(style.YearLeft)
	}

	t, err := time.ParseInLocation(style.Layout, line[start-n.YearOffsetAdjust:start-n.YearOffsetAdjust+len(style.Layout)], tz)
	if err != nil {
		return ParseError
	}
	return util.ToMS(t)
}
