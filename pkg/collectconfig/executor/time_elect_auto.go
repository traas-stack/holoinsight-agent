/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/timeparser"
	"time"
)

type (
	timeElectAutoParser struct {
		elect XElect
		w     *timeparser.TimeElectorWrapper
	}
)

func (e *timeElectAutoParser) Parse(ctx *LogContext) (int64, error) {
	line := ctx.GetLine()
	if e.elect != nil {
		s, err := e.elect.ElectString(ctx)
		if err != nil {
			return TimeParseError, err
		}
		line = s
	}
	if e.w == nil {
		e.w = timeparser.DetectTimeElectFromLine(line)
	}
	if e.w == nil {
		return TimeParseError, nil
	}
	tz := ctx.tz
	if tz == nil {
		tz = time.Local
	}
	ms := e.w.Elector.Parse(e.w.Style, line, tz)
	if ms == timeparser.ParseError {
		// style may change, need reparse
	}
	return ms, nil
}
