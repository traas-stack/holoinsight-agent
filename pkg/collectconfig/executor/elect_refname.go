/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/oliveagle/jsonpath"
	"github.com/spf13/cast"
	"strings"
)

type (
	xElectRefName struct {
		name     string
		jsonpath *jsonpath.Compiled
		err      error
	}
)

func (x *xElectRefName) Init() {
	if strings.HasPrefix(x.name, "$") {
		x.jsonpath, x.err = jsonpath.Compile(x.name)
	}
}

func (x *xElectRefName) Elect(ctx *LogContext) (interface{}, error) {
	if x.err != nil {
		return nil, x.err
	}
	if x.jsonpath != nil {
		y, err := x.jsonpath.Lookup(ctx.columnMap)
		if err != nil && len(ctx.logTags) > 0 {
			y, err = x.jsonpath.Lookup(ctx.logTags)
		}
		return y, err
	}
	return ctx.GetColumnByName(x.name)
}

func (x *xElectRefName) ElectString(ctx *LogContext) (string, error) {
	s, err := x.Elect(ctx)
	if err != nil {
		return "", nil
	}
	return cast.ToString(s), nil
}

func (x *xElectRefName) ElectNumber(ctx *LogContext) (float64, error) {
	c, err := x.Elect(ctx)
	if err != nil {
		return 0, nil
	}
	return cast.ToFloat64(c), nil
}
