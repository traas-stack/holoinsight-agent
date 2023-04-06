package executor

import (
	"errors"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
)

var ErrIndexOutOfBound = errors.New("index ouf of bound")

type (
	// xSubstringV1Filter
	xSubstringV1Filter struct {
		conf *collectconfig.TransformFilterSubstringV1
	}
)

func (x *xSubstringV1Filter) Init() error {
	return nil
}

func (x *xSubstringV1Filter) Filter(ctx *LogContext) (interface{}, error) {
	str := cast.ToString(ctx.contextValue)

	begin := x.conf.Begin
	if begin < 0 {
		begin = 0
	}

	if begin >= len(str) {
		if x.conf.EmptyIfError {
			return "", nil
		}
		return nil, ErrIndexOutOfBound
	}

	end := x.conf.End
	if end < 0 {
		end = len(str)
	}

	if end > len(str) {
		end = len(str)
	}

	if begin < end {
		return str[begin:end], nil
	}
	if x.conf.EmptyIfError {
		return "", nil
	}
	return nil, ErrIndexOutOfBound
}
