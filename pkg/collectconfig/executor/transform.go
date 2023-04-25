/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

type (
	XTransform interface {
		Transform(ctx *LogContext, v interface{}) (interface{}, error)
	}
	XPipeline interface {
		XTransform
	}
)
