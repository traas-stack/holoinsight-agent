/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import "errors"

type (
	xElectRefMeta struct {
		value string
	}
)

func (x *xElectRefMeta) Init() {
}

func (x *xElectRefMeta) Elect(ctx *LogContext) (interface{}, error) {
	return x.value, nil
}

func (x *xElectRefMeta) ElectString(ctx *LogContext) (string, error) {
	return x.value, nil
}

func (x *xElectRefMeta) ElectNumber(ctx *LogContext) (float64, error) {
	return 0, errors.New("unsupported")
}
