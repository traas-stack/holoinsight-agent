package ping

import (
	"encoding/json"
	"errors"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/ping"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/standard/providers"
	"github.com/traas-stack/holoinsight-agent/pkg/telegraf"
)

func init() {
	providers.Register("telegraf_ping", func(task *collecttask.CollectTask) (api.Input, error) {
		return parseInputPlugin(task)
	})
}

func parseInputPlugin(task *collecttask.CollectTask) (api.Input, error) {
	telegrafInput := inputs.Inputs["ping"]()
	ping := telegrafInput.(*ping.Ping)
	ping.Log = logger.ZapLogger.InfoS
	switch task.Target.Type {
	case collecttask.TargetPod:
		ping.Urls = []string{task.Target.GetIP()}
	case collecttask.TargetNone:
		if err := json.Unmarshal(task.Config.Content, ping); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported target type: " + task.Target.Type)
	}
	return telegraf.NewInputAdapter(telegrafInput), nil
}
