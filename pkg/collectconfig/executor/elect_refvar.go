/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import "github.com/spf13/cast"

type (
	xRefVar struct {
		name string
	}
)

func (x *xRefVar) Init() {
}

func (x *xRefVar) Elect(ctx *LogContext) (interface{}, error) {
	if ctx.vars == nil {
		return nil, nil
	}
	return ctx.vars[x.name], nil
}

func (x *xRefVar) ElectString(ctx *LogContext) (string, error) {
	if ctx.vars == nil {
		return "", nil
	}
	return cast.ToStringE(ctx.vars[x.name])
}

func (x *xRefVar) ElectNumber(ctx *LogContext) (float64, error) {
	if ctx.vars == nil {
		return 0, nil
	}
	return cast.ToFloat64E(ctx.vars[x.name])
}
