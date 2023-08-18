/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"time"
)

type (
	timeElectGolangLayoutParser struct {
		elect  XElect
		layout string
		tz     *time.Location
	}
)

func (e *timeElectGolangLayoutParser) Parse(ctx *LogContext) (int64, error) {
	s, err := e.elect.ElectString(ctx)
	if err != nil {
		return TimeParseError, err
	}

	tz := ctx.tz
	if tz == nil {
		tz = e.tz
	}

	parsed, err := time.ParseInLocation(e.layout, s, tz)
	if err != nil {
		return TimeParseError, fmt.Errorf("str=[%s] layout=[%s]: %v", s, e.layout, err)
	}
	return util.ToMS(parsed), nil
}
