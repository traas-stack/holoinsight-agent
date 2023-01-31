package executor

import (
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"
	"testing"
)

func TestTimeParse(t *testing.T) {
	parser, err := parseTimeParser(&collectconfig.TimeConf{
		Type: "auto",
	})
	if err != nil {
		panic(err)
	}
	ts, err := parser.Parse(&LogContext{log: &LogGroup{
		Line: "2022-04-22 14:50:31.000 process_lines,160",
	}})
	if err != nil {
		panic(err)
	}
	fmt.Println(ts)
}
