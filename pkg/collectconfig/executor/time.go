package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"time"
)

const (
	TypeAuto                 = "auto"
	TypeProcessTime          = "processTime"
	TypeElect                = "elect"
	FormatUnix               = "unix"
	FormatUnixMilli          = "unixMilli"
	FormatGolangLayout       = "golangLayout"
	timeFormatUnknown  uint8 = iota
	timeFormatUnix
	timeFormatUnixMilli
	timeFormatGolangLayout
	TimeParseError int64 = -2
)

type (
	TimeParser interface {
		Parse(*LogContext) (int64, error)
	}
)

func parseTimeParser(timeConf *collectconfig.TimeConf) (TimeParser, error) {
	if timeConf == nil {
		return &processTimeParser{}, nil
	}

	switch timeConf.Type {
	case TypeAuto:
		// ???
		return &autoParser{}, nil
	case TypeProcessTime:
		return &processTimeParser{}, nil
	case TypeElect:
		elect, err := parseElect(timeConf.Elect)
		if err != nil {
			return nil, err
		}
		tz := time.Local
		switch timeConf.Format {
		case FormatUnix:
			return &electParser{
				elect:  elect,
				format: timeFormatUnix,
			}, nil
		case FormatUnixMilli:
			return &electParser{
				elect:  elect,
				format: timeFormatUnixMilli,
			}, nil
		case FormatGolangLayout:
			if timeConf.Timezone != "" {
				tz0, err := time.LoadLocation(timeConf.Timezone)
				if err != nil {
					return nil, err
				}
				tz = tz0
			}

			_, err = time.ParseInLocation(timeConf.Layout,
				time.Now().Format(timeConf.Layout),
				time.Local)
			if err != nil {
				return nil, errors.New("invalid golang time layout " + timeConf.Layout)
			}

			return &electParser{
				elect:  elect,
				format: timeFormatGolangLayout,
				layout: timeConf.Layout,
				tz:     tz,
			}, nil
		default:
			return nil, errors.New("unsupported timeConf format " + timeConf.Format)
		}
	default:
		return nil, errors.New("unsupported timeConf")
	}
}
