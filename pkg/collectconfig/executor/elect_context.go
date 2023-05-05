/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import "github.com/spf13/cast"

type (
	// xElectContext elects value from LogContext.contextValue
	xElectContext struct{}
)

var xElectContextInstance = &xElectContext{}

func (x *xElectContext) Init() {
}

func (x *xElectContext) Elect(ctx *LogContext) (interface{}, error) {
	return ctx.contextValue, nil
}

func (x *xElectContext) ElectString(ctx *LogContext) (string, error) {
	return cast.ToStringE(ctx.contextValue)
}

func (x *xElectContext) ElectNumber(ctx *LogContext) (float64, error) {
	return cast.ToFloat64E(ctx.contextValue)
}
