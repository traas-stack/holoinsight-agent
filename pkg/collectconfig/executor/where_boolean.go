package executor

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"
)

type (
	xAnd []XWhere
	xOr  []XWhere
	xNot struct {
		sub XWhere
	}
)

func parseAndOr(and []*collectconfig.Where, isAnd bool) (XWhere, error) {
	if len(and) == 1 {
		return parseWhere(and[0])
	}

	a := make([]XWhere, 0, len(and))
	for _, sub := range and {
		subW, err := parseWhere(sub)
		if err != nil {
			return nil, err
		}
		a = append(a, subW)
	}

	if isAnd {
		return xAnd(a), nil
	} else {
		return xOr(a), nil
	}
}

func (x xAnd) Test(ctx *LogContext) (ret bool, _ error) {
	we := ctx.whereEvent
	if we != nil {
		we.Name = "and"
		defer func() {
			ctx.whereEvent = we
			we.Result = ret
		}()
	}

	for _, w := range x {
		if we != nil {
			ctx.whereEvent = we.AddChild()
		}
		b, err := w.Test(ctx)
		if err != nil {
			return false, err
		}
		if !b {
			return false, nil
		}
	}
	return true, nil
}

func (x xOr) Test(ctx *LogContext) (ret bool, _ error) {
	we := ctx.whereEvent
	if we != nil {
		we.Name = "or"
		defer func() {
			ctx.whereEvent = we
			we.Result = ret
		}()
	}

	for _, w := range x {
		if we != nil {
			ctx.whereEvent = we.AddChild()
		}
		b, err := w.Test(ctx)
		if err != nil {
			return false, err
		}
		if b {
			return true, nil
		}
	}
	return false, nil
}

func (x *xNot) Test(ctx *LogContext) (ret bool, _ error) {
	we := ctx.whereEvent
	if we != nil {
		we.Name = "not"
		defer func() {
			ctx.whereEvent = we
			we.Result = ret
		}()
		ctx.whereEvent = we.AddChild()
	}

	b, err := x.sub.Test(ctx)
	if err != nil {
		return false, err
	}
	return !b, nil
}
