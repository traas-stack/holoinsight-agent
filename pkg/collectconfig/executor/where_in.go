package executor

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"strings"
)

type (
	xIn struct {
		elect      XElect
		values     []string
		ignoreCase bool
	}
)

func (x *xIn) Test(ctx *LogContext) (ret bool, _ error) {
	we := ctx.whereEvent
	if we != nil {
		we.Name = "in"
		defer func() {
			we.Result = ret
		}()
	}

	str, err := x.elect.ElectString(ctx)
	if err != nil {
		return false, err
	}
	if x.ignoreCase {
		str = strings.ToLower(str)
	}
	return util.StringSliceContains(x.values, str), nil
}

func parseIn(r *collectconfig.MIn) (XWhere, error) {
	elect, err := parseElect(r.Elect)
	if err != nil {
		return nil, err
	}
	in := &xIn{
		elect:  elect,
		values: r.Values,
	}
	if r.IgnoreCase {
		for i := range in.values {
			in.values[i] = strings.ToLower(in.values[i])
		}
	}
	return in, nil
}
