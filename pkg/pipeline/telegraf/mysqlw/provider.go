package mysqlw

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/telegraf/providers"
	"github.com/influxdata/telegraf/plugins/inputs/mysql"
)

func init() {
	providers.Register("MysqlTask", parse)
}

func parse(task *collecttask.CollectTask) (interface{}, error) {
	conf := &MySQLConf{}
	err := json.Unmarshal(task.Config.Content, conf)
	if err != nil {
		return nil, err
	}

	var servers []string
	for _, conf := range conf.Confs {
		if conf.Port <= 0 {
			conf.Port = 3306
		}
		server := fmt.Sprintf("%s:%s@tcp(%s:%d)/", conf.User, conf.Password, conf.Host, conf.Port)
		servers = append(servers, server)
	}
	if len(servers) == 0 {
		return nil, errors.New("empty servers")
	}
	return &mysql.Mysql{
		Servers: servers,
	}, nil
}
