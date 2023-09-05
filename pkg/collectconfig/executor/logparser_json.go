/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"encoding/json"
	"strings"
)

type (
	jsonParser struct {
	}
)

func (j *jsonParser) Parse(ctx *LogContext) error {
	if len(ctx.columnMap) == 0 && strings.HasPrefix(ctx.GetLine(), "{") {
		return json.Unmarshal([]byte(ctx.GetLine()), &ctx.columnMap)
	}
	return nil
}
