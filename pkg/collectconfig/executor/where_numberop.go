package executor

import "github.com/traas-stack/holoinsight-agent/pkg/collectconfig"

type (
	xNumberOp struct {
		elect XElect
		lt    float64
		lte   float64
		gt    float64
		gte   float64
		eqi   int64
		nei   int64
		// ltc表示lt是可用的
		ltc  bool
		ltec bool
		gtc  bool
		gtec bool
		eqic bool
		neic bool
	}
)

func (x *xNumberOp) Test(ctx *LogContext) (ret bool, _ error) {
	we := ctx.whereEvent
	if we != nil {
		we.Name = "numberOp"
		defer func() {
			we.Result = ret
		}()
	}

	v, err := x.elect.ElectNumber(ctx)
	if err != nil {
		return false, err
	}

	if x.ltc && !(v < x.lt) {
		return false, nil
	}

	if x.ltec && !(v <= x.lte) {
		return false, nil
	}

	if x.gtc && !(v > x.gt) {
		return false, nil
	}

	if x.gtec && !(v >= x.gte) {
		return false, nil
	}

	if x.eqic && !(int64(v) == x.eqi) {
		return false, nil
	}

	if x.neic && !(int64(v) != x.nei) {
		return false, nil
	}

	return true, nil
}

func parseNumberOp(op *collectconfig.MNumberOp) (XWhere, error) {
	elect, err := parseElect(op.Elect)
	if err != nil {
		return nil, err
	}
	xop := &xNumberOp{
		elect: elect,
	}
	if op.Lt != nil {
		xop.lt = *op.Lt
		xop.ltc = true
	}
	if op.Lte != nil {
		xop.lte = *op.Lte
		xop.ltec = true
	}
	if op.Gt != nil {
		xop.gt = *op.Gt
		xop.gtc = true
	}
	if op.Gte != nil {
		xop.gte = *op.Gte
		xop.gtec = true
	}
	return xop, nil
}
