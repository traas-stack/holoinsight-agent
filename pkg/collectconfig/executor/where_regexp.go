package executor

import (
	"errors"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"
	"regexp"
)

type (
	xRegexp struct {
		elect     XElect
		regexp    *regexp.Regexp
		multiline bool
	}
)

func (x *xRegexp) Test(ctx *LogContext) (ret bool, _ error) {
	we := ctx.whereEvent
	if we != nil {
		we.Name = "regexp"
		defer func() {
			we.Result = ret
		}()
	}

	if x.multiline {
		for _, line := range ctx.log.Lines {
			if x.regexp.MatchString(line) {
				return true, nil
			}
		}
		return false, nil
	} else {
		s, err := x.elect.ElectString(ctx)
		if err != nil {
			return false, err
		}
		return x.regexp.MatchString(s), nil
	}
}

func parseRegexp(r *collectconfig.MRegexp) (XWhere, error) {
	elect, err := parseElect(r.Elect)
	if err != nil {
		return nil, err
	}
	compile, err := regexp.Compile(r.Expression)
	if err != nil {
		return nil, err
	}
	if r.Multiline && r.Elect.Type != collectconfig.EElectLine {
		return nil, errors.New("elect.type must be 'line' when using regexp.multiline mode")
	}
	return &xRegexp{
		elect:     elect,
		regexp:    compile,
		multiline: r.Multiline,
	}, nil
}
