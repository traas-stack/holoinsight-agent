/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/spf13/cast"
)

type (
	timeElectUnixMilliParser struct {
		elect XElect
	}
)

func (e *timeElectUnixMilliParser) Parse(ctx *LogContext) (int64, error) {
	s, err := e.elect.ElectString(ctx)
	if err != nil {
		return TimeParseError, err
	}
	i64, err := cast.ToInt64E(s)
	if err != nil {
		return TimeParseError, err
	}
	return i64, nil
}
