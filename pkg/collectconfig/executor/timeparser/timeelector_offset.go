/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package timeparser

import "time"

type (
	offsetTimeElector struct {
		maxOffset int
		layouts   []string
	}
)

func (o *offsetTimeElector) ParseTimeStyle(line string) *TimeStyle {
	// 这里是关键所在
	for i := 0; i <= o.maxOffset; i++ {
		for _, layout := range o.layouts {
			if i+len(layout) < len(line) {
				_, err := time.ParseInLocation(layout, line[i:i+len(layout)], time.Local)
				if err == nil {
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
