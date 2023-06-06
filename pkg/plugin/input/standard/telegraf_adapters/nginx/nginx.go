/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package nginx

import (
	"encoding/json"
	"errors"
	"fmt"
	telegraf2 "github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/plugins/inputs/nginx"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/standard/providers"
	"github.com/traas-stack/holoinsight-agent/pkg/telegraf"
	"strings"
	"time"
)

const (
	defaultTimeout = 3 * time.Second
	defaultPort    = 80
)

type (
	Conf struct {
		Urls []string `json:"urls,omitempty"`
		Port int      `json:"port,omitempty"`
		Path string   `json:"path,omitempty"`
	}
)

func init() {
	providers.RegisterInputProvider("telegraf_nginx", func(task *collecttask.CollectTask) (api.Input, error) {
		conf := &Conf{}
		if err := json.Unmarshal(task.Config.Content, conf); err != nil {
			return nil, err
		}
		var telegrafInput telegraf2.Input
		switch strings.ToLower(task.Target.Type) {
		case collecttask.TargetPod:
			if conf.Port <= 0 {
				conf.Port = defaultPort
			}
			url := fmt.Sprintf("http://%s:%d%s", task.Target.GetIP(), conf.Port, conf.Path)
			telegrafInput = &nginx.Nginx{
				Urls:            []string{url},
				ResponseTimeout: config.Duration(defaultTimeout),
			}
		case collecttask.TargetNone:
			telegrafInput = &nginx.Nginx{
				Urls:            conf.Urls,
				ResponseTimeout: config.Duration(defaultTimeout),
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
