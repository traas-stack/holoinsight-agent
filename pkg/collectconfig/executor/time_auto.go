/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/timeparser"
	"time"
)

type (
	autoParser struct {
		w *timeparser.TimeElectorWrapper
	}
)

func (a *autoParser) Parse(context *LogContext) (int64, error) {
	if a.w == nil {
		a.w = timeparser.DetectTimeElectFromLine(context.GetLine())
	}
	if a.w == nil {
		return TimeParseError, nil
	}
	tz := context.tz
	if tz == nil {
		tz = time.Local
	}
	ms := a.w.Elector.Parse(a.w.Style, context.GetLine(), tz)
	if ms == timeparser.ParseError {
		// style may change, need reparse
	}
	return ms, nil
}
