package executor

import "github.com/spf13/cast"

type (
	xElectRefIndex struct {
		index int
	}
)

func (x *xElectRefIndex) Init() {
}

func (x *xElectRefIndex) ElectString(ctx *LogContext) (string, error) {
	return ctx.GetColumnByIndex(x.index)
}

func (x *xElectRefIndex) ElectNumber(ctx *LogContext) (float64, error) {
	c, err := x.Elect(ctx)
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(c)
}

func (x *xElectRefIndex) Elect(ctx *LogContext) (interface{}, error) {
	return ctx.GetColumnByIndex(x.index)
}
