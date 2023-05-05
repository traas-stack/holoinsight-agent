/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"strings"
)

type (
	xAppendV1Filter struct {
		conf *collectconfig.TransformFilterAppendV1
	}
)

func (x *xAppendV1Filter) Init() error {
	return nil
}

func (x *xAppendV1Filter) Filter(ctx *LogContext) (interface{}, error) {
	str := cast.ToString(ctx.contextValue)
	if !x.conf.AppendIfMissing || !strings.HasSuffix(str, x.conf.Value) {
		str += x.conf.Value
	}
	return str, nil
}
