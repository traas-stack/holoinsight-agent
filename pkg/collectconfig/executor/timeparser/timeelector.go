package timeparser

import (
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
	}
)

var type0 = &SimpleTimeElector{
	YearOffsetAdjust: 0,
	layouts: []string{
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02T15:04:05",
		"2006 Jan/02 15:04:05",
	},
}

var type1 = &SimpleTimeElector{
	YearOffsetAdjust: 7,
	layouts: []string{
		"02/Jan/2006 15:04:05",
		"Jan 02 2006 15:04:05",
	},
}

var type2 = &SimpleTimeElector{
	YearOffsetAdjust: 6,
	layouts: []string{
		"01/02/2006 15:04:05",
	},
}

var offset = &offsetTimeElector{
	maxOffset: 4,
	layouts: []string{
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02T15:04:05",
		"2006 Jan/02 15:04:05",
		"02/Jan/2006 15:04:05",
		"Jan 02 2006 15:04:05",
		"01/02/2006 15:04:05",
	},
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
