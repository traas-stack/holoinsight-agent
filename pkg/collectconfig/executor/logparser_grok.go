/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/vjeantet/grok"
	_ "github.com/vjeantet/grok"
)

type (
	grokParser struct {
		g          *grok.Grok
		expression string
	}
)

func (g *grokParser) Parse(ctx *LogContext) error {
	m, err := g.g.ParseTyped(g.expression, ctx.GetLine())
	if err != nil {
		return err
	}
	if len(m) == 0 {
		return LogParseNotMatched
	}
	ctx.columnMap = m
	return nil
}
