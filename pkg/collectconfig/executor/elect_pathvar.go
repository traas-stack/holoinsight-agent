/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import "github.com/spf13/cast"

type (
	xPathVar struct {
		name string
	}
)

func (x *xPathVar) Init() {
}

func (x *xPathVar) Elect(ctx *LogContext) (interface{}, error) {
	return x.ElectString(ctx)
}

func (x *xPathVar) ElectString(ctx *LogContext) (string, error) {
	if len(ctx.pathTags) > 0 {
		return ctx.pathTags[x.name], nil
	}
	return "", nil
}

func (x *xPathVar) ElectNumber(ctx *LogContext) (float64, error) {
	i, _ := x.Elect(ctx)
	return cast.ToFloat64(i), nil
}
