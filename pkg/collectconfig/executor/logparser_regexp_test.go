/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"testing"
)

func TestLogParserRegexp(t *testing.T) {
	logparser, err := parseLogParser(&collectconfig.FromLogParse{
		Where:     nil,
		Type:      "regexp",
		Separator: nil,
		Regexp: &collectconfig.LogParseRegexp{
			Expression: "^(?P<first>[a-z]+)-(?P<second>[0-9]+)$",
		},
		Grok: nil,
	})
	assert.NoError(t, err, "parseLogParser error")
	ctx := &LogContext{}
	ctx.log = &LogGroup{
		Line: "abc-123",
	}
	err = logparser.Parse(ctx)
	assert.NoError(t, err, "parse error")
	assert.Equal(t, "abc", ctx.columns[0])
	assert.Equal(t, "123", ctx.columns[1])
	assert.Equal(t, "abc", ctx.columnMap["first"])
	assert.Equal(t, "123", ctx.columnMap["second"])
}
