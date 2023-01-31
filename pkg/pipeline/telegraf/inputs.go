package telegraf

import (
	"errors"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collecttask"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/pipeline/telegraf/mysqlw"
	"github.com/TRaaSStack/holoinsight-agent/pkg/pipeline/telegraf/providers"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/dialcheckw"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/httpcheckw"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/jvm"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/springboot"
	"strings"
)

func trimType(t string) string {
	index := strings.LastIndexByte(t, '.')
	if index < 0 {
		return t
	}
	return t[index+1:]
}

func parseInput(task *collecttask.CollectTask) (interface{}, error) {
	trimmedType := trimType(task.Config.Type)

	if provider, ok := providers.Get(trimmedType); ok {
		return provider(task)
	}

	switch trimmedType {
	default:
		return nil, errors.New("unsupported type " + task.Config.Type)
	}
}
