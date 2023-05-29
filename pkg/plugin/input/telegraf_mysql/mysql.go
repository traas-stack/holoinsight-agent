package telegraf_mysql

import (
	"encoding/json"
	"errors"
	"fmt"
	telegraf2 "github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs/mysql"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/telegraf/providers"
	"github.com/traas-stack/holoinsight-agent/pkg/telegraf"
)

type (
	Conf struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
		Port     int    `json:"port,omitempty"`
	}
)

func init() {
	providers.Register("telegraf_mysql", func(task *collecttask.CollectTask) (interface{}, error) {
		conf := &Conf{}
		if err := json.Unmarshal(task.Config.Content, conf); err != nil {
			return nil, err
		}

		var telegrafInput telegraf2.Input
		switch task.Target.Type {
		case collecttask.TargetPod:
			if conf.Port <= 0 {
				conf.Port = 3306
			}
			server := fmt.Sprintf("%s:%s@tcp(%s:%d)/?tls=false", conf.Username, conf.Password, task.Target.GetIP(), conf.Port)
			telegrafInput = &mysql.Mysql{
				Servers: []string{server},
				Log:     logger.ZapLogger.InfoS,
			}
		case collecttask.TargetNone:
			telegrafInput = &mysql.Mysql{
				Log: logger.ZapLogger.InfoS,
			}
			if err := json.Unmarshal(task.Config.Content, telegrafInput); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("unsupported target type: " + task.Target.Type)
		}
		return telegraf.NewInputAdapter(telegrafInput), nil
	})
}
