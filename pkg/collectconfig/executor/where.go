package executor

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"
)

type (
	XWhere interface {
		Test(ctx *LogContext) (bool, error)
	}
)

// TODO temp
func MustParseWhere(w *collectconfig.Where) XWhere {
	ret, err := parseWhere(w)
	if err != nil {
		panic(err)
	}
	return ret
}

func parseWhere(w *collectconfig.Where) (XWhere, error) {
	if w == nil {
		return nil, nil
	}
	if len(w.And) > 0 {
		return parseAndOr(w.And, true)
	}
	if len(w.Or) > 0 {
		return parseAndOr(w.Or, false)
	}
	if w.Not != nil {
		subW, err := parseWhere(w.Not)
		if err != nil {
			return nil, err
		}
		return &xNot{sub: subW}, nil
	}
	if w.Contains != nil {
		return parseContains(w.Contains)
	}
	if w.ContainsAny != nil {
		return parseContainsAny(w.ContainsAny)
	}

	if w.NumberOp != nil {
		return parseNumberOp(w.NumberOp)
	}

	if w.Regexp != nil {
		return parseRegexp(w.Regexp)
	}

	if w.In != nil {
		return parseIn(w.In)
	}

	return &xWhereAlwaysTrue{}, nil
}
