package executor

type (
	XTransform interface {
		Transform(ctx *LogContext, v interface{}) (interface{}, error)
	}
	XPipeline interface {
		XTransform
	}
)
