/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import "github.com/spf13/cast"

type (
	xElectWrap struct {
		inner     XElect
		transform XTransformFilter
	}
)

func (x *xElectWrap) Init() {
	x.inner.Init()
}

func (x *xElectWrap) Elect(ctx *LogContext) (interface{}, error) {
	v, err := x.inner.Elect(ctx)
	if err != nil {
		return v, err
	}

	if x.transform != nil {
		ctx2 := &LogContext{contextValue: v}
		_, err = x.transform.Filter(ctx2)
		return ctx2.contextValue, err
	}

	return v, nil
}

func (x *xElectWrap) ElectString(ctx *LogContext) (string, error) {
	v, err := x.inner.ElectString(ctx)
	if err != nil {
		return v, err
	}

	if x.transform != nil {
		ctx2 := &LogContext{contextValue: v}
		_, err = x.transform.Filter(ctx2)
		return cast.ToString(ctx2.contextValue), err
	}

	return v, nil
}

func (x *xElectWrap) ElectNumber(ctx *LogContext) (float64, error) {
	v, err := x.inner.Elect(ctx)
	if err != nil {
		return 0, err
	}

	if x.transform != nil {
		ctx2 := &LogContext{contextValue: v}
		_, err = x.transform.Filter(ctx2)
		if err != nil {
			return cast.ToFloat64(ctx2.contextValue), err
		}
		return cast.ToFloat64E(ctx2.contextValue)
	}

	return cast.ToFloat64E(v)
}
