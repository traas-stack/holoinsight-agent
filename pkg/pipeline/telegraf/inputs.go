/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package telegraf

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	_ "github.com/traas-stack/holoinsight-agent/pkg/pipeline/telegraf/mysqlw"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/telegraf/providers"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/dialcheckw"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/httpcheckw"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/jvm"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/nvidia_smi"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/springboot"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/telegraf_mongodb"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/telegraf_mysql"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/telegraf_nginx"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/telegraf_ping"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/telegraf_redis"
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
