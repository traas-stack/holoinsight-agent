package executor

import (
	"regexp"
)

type (
	regexpParser struct {
		reg *regexp.Regexp
	}
)

func (r *regexpParser) Parse(ctx *LogContext) error {
	ss := r.reg.FindStringSubmatch(ctx.GetLine())
	if ss == nil {
		return LogParseNotMatched
	}
	names := r.reg.SubexpNames()
	m := make(map[string]interface{}, len(ss)-1)
	columns := make([]string, 0, len(ss)-1)
	for i := 1; i < len(ss); i++ {
		m[names[i]] = ss[i]
		columns = append(columns, ss[i])
	}
	ctx.columns = columns
	ctx.columnMap = m
	return nil
}
