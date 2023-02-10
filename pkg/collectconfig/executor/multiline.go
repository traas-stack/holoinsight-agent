package executor

import (
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
)

const (
	multilineWhatPrevious uint8 = iota
	multilineWhatNext
	defaultMaxLinesPerGroup = 1024
	maxLinesPerGroup        = 4096
)

type (
	xMultiline struct {
		where    XWhere
		what     uint8
		maxLines int
	}
)

func parseMultiline(cfg *collectconfig.FromLogMultiline) (*xMultiline, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}
	if cfg.Where == nil {
		return nil, errors.New("multiline.start is nil")
	}
	what := multilineWhatPrevious
	switch cfg.What {
	case "previous":
		what = multilineWhatPrevious
	case "next":
		what = multilineWhatNext
	default:
		return nil, fmt.Errorf("invalid what [%s]", cfg.What)
	}
	w, err := parseWhere(cfg.Where)
	if err != nil {
		return nil, err
	}
	maxLines := cfg.MaxLines
	if maxLines <= 0 {
		maxLines = defaultMaxLinesPerGroup
	}
	if maxLines > maxLinesPerGroup {
		maxLines = maxLinesPerGroup
	}
	return &xMultiline{
		where:    w,
		what:     what,
		maxLines: cfg.MaxLines,
	}, nil
}
