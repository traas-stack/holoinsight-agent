package executor

import (
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
)

type (
	xMappingV1Filter struct {
		conf *collectconfig.TransformFilterMappingV1
	}
)

func (x *xMappingV1Filter) Init() error {
	return nil
}

func (x *xMappingV1Filter) Filter(ctx *LogContext) (interface{}, error) {
	if x.conf.Mappings == nil {
		return ctx.contextValue, nil
	}
	str := cast.ToString(ctx.contextValue)
	if v, ok := x.conf.Mappings[str]; ok {
		return v, nil
	} else {
		if x.conf.DefaultValue != "" {
			return x.conf.DefaultValue, nil
		}
		return ctx.contextValue, nil
	}
}
