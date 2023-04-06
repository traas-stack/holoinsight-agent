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
	return x.value, nil
}
