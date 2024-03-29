/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

type (
	// xConstFilter always returns a const value
	xConstFilter struct {
		value string
	}
)

func (x *xConstFilter) Init() error {
	return nil
}

func (x *xConstFilter) Filter(ctx *LogContext) (interface{}, error) {
	if x.value != "" {
		return x.value, nil
	}
	return ctx.contextValue, nil
}
