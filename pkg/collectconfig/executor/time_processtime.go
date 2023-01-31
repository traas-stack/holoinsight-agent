package executor

import "github.com/TRaaSStack/holoinsight-agent/pkg/util"

type (
	processTimeParser struct {
	}
)

func (a *processTimeParser) Parse(context *LogContext) (int64, error) {
	return util.CurrentMS(), nil
}
