/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import "errors"

type (
	// 引用原始日志行
	xElectLine struct {
		// 用于多行模式, 第几行
		index int
	}
)

var (
	electFirstLine = &xElectLine{}
)

func (x *xElectLine) Init() {
}

func (x *xElectLine) ElectString(ctx *LogContext) (string, error) {
	if x.index == 0 {
		return ctx.GetLine(), nil
	}
	if x.index < len(ctx.log.Lines) {
		return ctx.log.Lines[x.index], nil
	}
	return "", nil
}

func (x *xElectLine) ElectNumber(ctx *LogContext) (float64, error) {
	return 0, errors.New("unsupported")
}

func (x *xElectLine) Elect(ctx *LogContext) (interface{}, error) {
	return ctx.GetLine(), nil
}
