/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

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
	FormatAuto               = "auto"
	TimeParseError     int64 = -2
)

type (
	TimeParser interface {
		Parse(*LogContext) (int64, error)
	}
)

func parseTimeParser(timeConf *collectconfig.TimeConf) (TimeParser, error) {
	if timeConf == nil {
		return &timeElectAutoParser{elect: electFirstLine}, nil
	}

	switch timeConf.Type {
	case TypeAuto:
		return &timeElectAutoParser{elect: electFirstLine}, nil
	case TypeProcessTime:
		return &timeProcessTimeParser{}, nil
	case TypeElect:
		elect, err := parseElect(timeConf.Elect)
		if err != nil {
			return nil, err
		}
		switch timeConf.Format {
		case FormatAuto:
			return &timeElectAutoParser{elect: elect}, nil
		case FormatUnix:
			return &timeElectUnixParser{elect: elect}, nil
		case FormatUnixMilli:
			return &timeElectUnixMilliParser{elect: elect}, nil
		case FormatGolangLayout:
			tz := time.Local
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

			return &timeElectGolangLayoutParser{elect: elect, layout: timeConf.Layout, tz: tz}, nil
		default:
			return nil, errors.New("unsupported timeConf format " + timeConf.Format)
		}
	default:
		return nil, errors.New("unsupported timeConf")
	}
}
