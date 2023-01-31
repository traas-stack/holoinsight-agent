package executor

import (
	"errors"
	"github.com/spf13/cast"
	"regexp"
)

type (
	xRegexpElect struct {
		r     *regexp.Regexp
		index int
	}
)

func (x *xRegexpElect) Init() {
}

func (x *xRegexpElect) Elect(ctx *LogContext) (interface{}, error) {
	return x.ElectString(ctx)
}

func (x *xRegexpElect) ElectString(ctx *LogContext) (string, error) {
	ss := x.r.FindStringSubmatch(ctx.GetLine())
	if x.index < len(ss) {
		return ss[x.index], nil
	}
	return "", errors.New("elect error")
}

func (x *xRegexpElect) ElectNumber(ctx *LogContext) (float64, error) {
	v, err := x.Elect(ctx)
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(v)
}
