package executor

import (
	"errors"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"time"
)

type (
	electParser struct {
		elect  XElect
		tz     *time.Location
		format uint8
		layout string
	}
)

func (t *electParser) Parse(ctx *LogContext) (int64, error) {
	c, err := t.elect.Elect(ctx)
	if err != nil {
		return TimeParseError, err
	}
	// TODO format 预编译
	ts := int64(-1)
	switch t.format {
	case timeFormatUnix:
		// unix 秒
		i64, _ := cast.ToInt64E(c)
		ts = util.ToMS(time.Unix(i64, 0))
	case timeFormatUnixMilli:
		i64, _ := cast.ToInt64E(c)
		ts = util.ToMS(time.UnixMilli(i64))
	case timeFormatGolangLayout:
		s, err := cast.ToStringE(c)
		if err != nil {
			return TimeParseError, err
		}
		parsed, err := time.ParseInLocation(t.layout, s, t.tz)
		if err != nil {
			return -2, err
		}
		return util.ToMS(parsed), nil
	default:
		return TimeParseError, errors.New("unsupported")
	}
	return ts, nil
}
