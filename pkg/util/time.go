/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"errors"
	"github.com/spf13/cast"
	"time"
)

func CurrentMS() int64 {
	return time.Now().UnixNano() / 1000000
}
func ToMS(t time.Time) int64 {
	return t.UnixNano() / 1e6
}

func BuildPeriod(interval int64) int64 {
	timestamp := time.Now().UnixNano() / 1000000
	ret := timestamp / interval * interval
	return ret
}

// NextDelay 返回下一个时间窗口的delay
func NextDelay(interval time.Duration) time.Duration {
	return NextDelayFrom(time.Now(), interval)
}

func NextDelayFrom(now time.Time, interval time.Duration) time.Duration {
	return now.Truncate(interval).Add(interval).Sub(now)
}

// ParseDuration parse any object that like a duration str to duration
// for example:
// "100ms" -> 100ms
// 100 -> 100ms
// 100.0 -> 100ms
func ParseDuration(d interface{}) (time.Duration, error) {
	if s, ok := d.(string); ok {
		if p, err := time.ParseDuration(s); err == nil {
			return p, nil
		}
	}

	f64, err := cast.ToFloat64E(d)
	if err != nil {
		return 0, err
	}
	i64 := int64(f64)
	if i64 < 0 {
		return 0, errors.New("duration < <0")
	}
	return time.Duration(i64) * time.Millisecond, nil
}

func ParseDurationDefault(s string, d time.Duration) time.Duration {
	if d2, err := time.ParseDuration(s); err == nil {
		return d2
	}
	return d
}

func TimeTruncateToDay(t time.Time) time.Time {
	t = t.Truncate(time.Hour)
	return t.Add(-time.Duration(t.Hour()) * time.Hour)
}
