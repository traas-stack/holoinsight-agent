package executor

import "github.com/spf13/cast"

type (
	xElectRefName struct {
		name string
	}
)

func (x *xElectRefName) Init() {
}

func (x *xElectRefName) Elect(ctx *LogContext) (interface{}, error) {
	return ctx.GetColumnByName(x.name)
}

func (x *xElectRefName) ElectString(ctx *LogContext) (string, error) {
	s, err := ctx.GetColumnByName(x.name)
	if err != nil {
		return "", err
	}
	return cast.ToStringE(s)
}

func (x *xElectRefName) ElectNumber(ctx *LogContext) (float64, error) {
	c, err := x.Elect(ctx)
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(c)
}
