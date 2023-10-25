/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/spf13/cast"
	"net/url"
	"strings"
)

type (
	// xCleanUrlV1Filter
	xCleanUrlV1Filter struct {
	}
)

func (x *xCleanUrlV1Filter) Init() error {
	return nil
}

func (x *xCleanUrlV1Filter) Filter(ctx *LogContext) (interface{}, error) {
	u := cast.ToString(ctx.contextValue)

	if strings.HasPrefix(u, "http") {
		if u2, err := url.Parse(u); err == nil {
			u = u2.Path
		}
	} else {
		if i := strings.IndexByte(u, '?'); i >= 0 {
			u = u[:i]
		}
	}

	return u, nil
}
