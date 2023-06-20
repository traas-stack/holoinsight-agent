/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package timeparser

import (
	"time"
)

type (
	offsetTimeElector struct {
		maxOffset int
		layouts   []timeLayout
	}
	timeLayout struct {
		layout      string
		hasTimezone bool
	}
	sortByLayoutLength []timeLayout
)

func (o *offsetTimeElector) ParseTimeStyle(line string) *TimeStyle {
	// 这里是关键所在
	for i := 0; i <= o.maxOffset; i++ {
		for _, tl := range o.layouts {
			layout := tl.layout
			if tl.hasTimezone {
				zIndex := i + len(layout) - 6
				if zIndex < len(line) && line[zIndex] == 'Z' {
					s := line[i : zIndex+1]
					if _, err := time.ParseInLocation(layout, s, time.Local); err == nil {
						return &TimeStyle{
							YearLeftIndex: i,
							Layout:        layout,
							HasTimezone:   true,
						}
					}
				}
			}

			if i+len(layout) <= len(line) {
				if _, err := time.ParseInLocation(layout, line[i:i+len(layout)], time.Local); err == nil {
					return &TimeStyle{
						YearLeftIndex: i,
						Layout:        layout,
					}
				}
			}
		}
	}
	return nil
}

func (o *offsetTimeElector) Parse(style *TimeStyle, line string, tz *time.Location) int64 {
	if style.HasTimezone {
		zIndex := style.YearLeftIndex + len(style.Layout) - 6
		if zIndex < len(line) && line[zIndex] == 'Z' {
			if t, err := time.ParseInLocation(style.Layout, line[style.YearLeftIndex:zIndex+1], tz); err == nil {
				return t.UnixMilli()
			}
		}
	}

	minLen := style.YearLeftIndex + len(style.Layout)
	if len(line) < minLen {
		return ParseError
	}
	sub := line[style.YearLeftIndex:minLen]
	if tz == nil {
		tz = time.Local
	}
	t, err := time.ParseInLocation(style.Layout, sub, tz)
	if err != nil {
		return ParseError
	}
	return t.UnixMilli()
}

func (o *offsetTimeElector) containsLayout(layout string) bool {
	for _, tl := range o.layouts {
		if tl.layout == layout {
			return true
		}
	}
	return false
}

func (s sortByLayoutLength) Len() int {
	return len(s)
}

func (s sortByLayoutLength) Less(i, j int) bool {
	return len(s[i].layout) < len(s[j].layout)
}

func (s sortByLayoutLength) Swap(i, j int) {
	t := s[i]
	s[i] = s[j]
	s[j] = t
}
