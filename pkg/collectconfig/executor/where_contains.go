package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"strings"
)

type (
	xContains struct {
		elect      XElect
		value      []string
		multiline  bool
		ignoreCase bool
	}
)

func (x *xContains) Test(ctx *LogContext) (ret bool, _ error) {
	we := ctx.whereEvent
	if we != nil {
		we.Name = "contains"
		defer func() {
			we.Result = ret
		}()
	}

	if x.multiline {
		// TODO 此时 x.elect.type 必须是line
		for _, line := range ctx.log.Lines {
			if x.ignoreCase {
				line = strings.ToLower(line)
			}
			for _, str := range x.value {
				if strings.Contains(line, str) {
					return true, nil
				}
			}
		}
		return false, nil
	} else {
		v, err := x.elect.Elect(ctx)
		if err != nil {
			return false, err
		}
		if s, ok := v.(string); ok {
			if x.ignoreCase {
				s = strings.ToLower(s)
			}
			for _, str := range x.value {
				if strings.Contains(s, str) {
					return true, nil
				}
			}
		}
		return false, nil
	}
}

func parseContains(contains *collectconfig.MContains) (XWhere, error) {
	elect, err := parseElect(contains.Elect)
	if err != nil {
		return nil, err
	}
	if contains.Multiline && contains.Elect.Type != collectconfig.EElectLine {
		return nil, errors.New("elect.type must be 'line' when using multiline mode")
	}
	c := &xContains{
		elect:      elect,
		value:      []string{contains.Value},
		multiline:  contains.Multiline,
		ignoreCase: contains.IgnoreCase,
	}
	if contains.IgnoreCase {
		for i := range c.value {
			c.value[i] = strings.ToLower(c.value[i])
		}
	}
	return c, nil
}

func parseContainsAny(contains *collectconfig.MContainsAny) (XWhere, error) {
	elect, err := parseElect(contains.Elect)
	if err != nil {
		return nil, err
	}
	if contains.Multiline && contains.Elect.Type != collectconfig.EElectLine {
		return nil, errors.New("elect.type must be 'line' when using multiline mode")
	}
	c := &xContains{
		elect:      elect,
		value:      contains.Values,
		multiline:  contains.Multiline,
		ignoreCase: contains.IgnoreCase,
	}
	if contains.IgnoreCase {
		for i := range c.value {
			c.value[i] = strings.ToLower(c.value[i])
		}
	}
	return c, nil
}
