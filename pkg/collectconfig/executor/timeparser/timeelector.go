/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package timeparser

import (
	"os"
	"sort"
	"strings"
	"time"
)

const (
	TimeStyleNormal int = iota
	lineStart           = -1
	maxBytes            = 64
	ParseError          = -2
)

type (
	TimeElectorWrapper struct {
		Elector TimeElector
		Style   *TimeStyle
	}

	TimeElector interface {
		ParseTimeStyle(line string) *TimeStyle
		Parse(style *TimeStyle, line string, tz *time.Location) int64
	}
	TimeStyle struct {
		YearLeft             string
		YearLeftIndex        int
		Layout               string
		TimestampMillisMode  bool
		TimestampSecondsMode bool
		// HasTimezone is true if layout ends with 'Z07:00'
		HasTimezone bool
	}
)

var basicLayouts = []string{
	"2006-01-02 15:04:05",
	"2006/01/02 15:04:05",
	"2006-01-02T15:04:05",
}

var offset = &offsetTimeElector{
	maxOffset: 4,
	layouts: []timeLayout{
		{layout: time.UnixDate},
		{layout: time.ANSIC},
		{layout: "2006 Jan/02 15:04:05"},
		{layout: "02/Jan/2006 15:04:05"},
		{layout: "Jan 02 2006 15:04:05"},
		{layout: "01/02/2006 15:04:05"},
	},
}

func init() {
	for _, layout := range basicLayouts {
		// Add many layout variants

		// .000 matches .000 or ,111
		// Z07:00 matches Z or +07:00

		offset.layouts = append(offset.layouts, timeLayout{layout: layout + ".000 Z07:00"})
		offset.layouts = append(offset.layouts, timeLayout{layout: layout + ".000Z07:00"})
		offset.layouts = append(offset.layouts, timeLayout{layout: layout + ".000"})
		offset.layouts = append(offset.layouts, timeLayout{layout: layout + " Z07:00"})
		offset.layouts = append(offset.layouts, timeLayout{layout: layout + "Z07:00"})
		offset.layouts = append(offset.layouts, timeLayout{layout: layout})
	}

	if s := os.Getenv("TIME_LAYOUTS"); s != "" {
		ss := strings.Split(s, "|")
		for _, layout := range ss {
			layout = strings.TrimSpace(layout)
			if offset.containsLayout(layout) {
				continue
			}
			now := time.Now()
			parsed, err := time.Parse(layout, now.Format(layout))
			if err != nil {
				continue
			}
			if now.Truncate(time.Minute) != parsed.Truncate(time.Minute) {
				continue
			}
			offset.layouts = append(offset.layouts, timeLayout{layout: layout})
		}
	}

	for i := range offset.layouts {
		tl := &offset.layouts[i]
		if strings.HasSuffix(tl.layout, "Z07:00") {
			tl.hasTimezone = true
		}
	}

	sort.Sort(sort.Reverse(sortByLayoutLength(offset.layouts)))
}

var timestamp = &timestampTimeElector{}

func ParseTimeStyle(line string) (TimeElector, *TimeStyle) {
	ts := offset.ParseTimeStyle(line)
	if ts != nil {
		return offset, ts
	}

	ts = timestamp.ParseTimeStyle(line)
	if ts != nil {
		return timestamp, ts
	}

	return nil, nil
}

func DetectTimeElectFromLine(line string) *TimeElectorWrapper {
	te, ts := ParseTimeStyle(line)
	if te == nil {
		return nil
	}
	return &TimeElectorWrapper{
		Elector: te,
		Style:   ts,
	}
}
