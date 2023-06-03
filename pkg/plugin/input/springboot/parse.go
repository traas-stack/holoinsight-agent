/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package springboot

import (
	"encoding/json"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/standard/providers"
	"strings"
)

const (
	defaultConcurrency = 4
	defaultPort        = 8080
)

type (
	SpringBootConf struct {
		Conf struct {
			Port    int    `json:"port"`
			BaseUrl string `json:"baseUrl"`
		} `json:"conf"`
		RefMetas struct {
			App struct {
				Name string `json:"name"`
			} `json:"app"`
		} `json:"refMetas"`
	}
)

func init() {
	providers.RegisterInputProvider("springboottask", Parse)
}

func Parse(task *collecttask.CollectTask) (api.Input, error) {
	conf := &SpringBootConf{}
	err := json.Unmarshal(task.Config.Content, conf)
	if err != nil {
		return nil, err
	}
	port := conf.Conf.Port
	if port <= 0 {
		port = defaultPort
	}
	path := conf.Conf.BaseUrl
	if strings.HasPrefix(path, "/") {
		path = strings.TrimLeft(path, "/")
	}
	host := "localhost"
	switch task.Target.Type {
	case "localhost":
		host = "localhost"
	case "pod":
		host = task.Target.GetIP()
	}
	baseUrl := fmt.Sprintf("http://%s:%d/%s", host, port, path)
	return &SpringBoot{
		Concurrency: defaultConcurrency,
		BaseUrl:     baseUrl,
	}, nil
}
