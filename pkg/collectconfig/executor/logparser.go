/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/vjeantet/grok"
	"regexp"
	"strings"
)

var (
	LogParseNotMatched = errors.New("LogParseNotMatched")
)

type (
	LogParser interface {
		Parse(ctx *LogContext) error
	}
	separatorParser struct {
		sep string
	}
)

func (s *separatorParser) Parse(ctx *LogContext) error {
	ctx.columns = strings.Split(ctx.GetLine(), s.sep)
	return nil
}

func parseLogParser(cfg *collectconfig.FromLogParse) (LogParser, error) {
	if cfg == nil {
		return nil, nil
	}
	switch cfg.Type {
	case "none":
		return nil, nil
	case "separator":
		if cfg.Separator == nil {
			return nil, errors.New("parse.separator is nil")
		}
		if cfg.Separator.Separator == "" {
			return nil, errors.New("parse.separator.separator is empty")
		}
		return &separatorParser{
			sep: cfg.Separator.Separator,
		}, nil
	case "json":
		return &jsonParser{}, nil
	case "regexp":
		if cfg.Regexp == nil {
			return nil, errors.New("parse.regexp is nil")
		}
		reg, err := regexp.Compile(cfg.Regexp.Expression)
		if err != nil {
			return nil, err
		}
		return &regexpParser{
			reg: reg,
		}, nil
	case "grok":
		g, err := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
		if err != nil {
			return nil, err
		}
		return &grokParser{
			g:          g,
			expression: cfg.Grok.Expression,
		}, nil
	}
	return nil, nil
}
