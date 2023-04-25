/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package timeparser

import (
	"strconv"
	"time"
)

const (
	millisTimestampDemo  = "1668517987046"
	secondsTimestampDemo = "1668517987"
)

type (
	// timestamp style time elector
	timestampTimeElector struct {
	}
)

func (t *timestampTimeElector) ParseTimeStyle(line string) *TimeStyle {
	// 'len(const string)' will be compiled into const number. There is no performance penalty when running.

	// millis:
	if len(line) > len(millisTimestampDemo) {
		_, err := strconv.ParseInt(line[:len(millisTimestampDemo)], 10, 64)
		if err == nil {
			return &TimeStyle{TimestampMillisMode: true}
		}
	}

	// seconds:
	if len(line) > len(secondsTimestampDemo) {
		_, err := strconv.ParseInt(line[:len(secondsTimestampDemo)], 10, 64)
		if err == nil {
			return &TimeStyle{TimestampSecondsMode: true}
		}
	}

	return nil
}

func (t *timestampTimeElector) Parse(style *TimeStyle, line string, tz *time.Location) int64 {
	if style.TimestampMillisMode && len(line) > len(millisTimestampDemo) {
		i64, err := strconv.ParseInt(line[:len(millisTimestampDemo)], 10, 64)
		if err != nil {
			return ParseError
		}
		return i64
	}

	if style.TimestampSecondsMode && len(line) > len(secondsTimestampDemo) {
		i64, err := strconv.ParseInt(line[:len(secondsTimestampDemo)], 10, 64)
		if err != nil {
			return ParseError
		}
		return i64 * 1000
	}

	return ParseError
}
