package executor

type (
	xDiscardFilter struct {
	}
)

func (x *xDiscardFilter) Init() error {
	return nil
}

func (x *xDiscardFilter) Filter(ctx *LogContext) (interface{}, error) {
	return nil, nil
}
