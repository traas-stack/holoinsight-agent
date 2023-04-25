/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package telegraf

import (
	"encoding/json"
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/integration/base"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

func ParsePipeline(task *collecttask.CollectTask) (*Pipeline, error) {
	i, err := parseInput(task)
	if err != nil {
		return nil, err
	}
	if i == nil {
		return nil, errors.New("parseInput returns nil")
	}
	var out output.Output
	if appconfig.IsDev() {
		out, err = output.Parse("console", nil)
	} else {
		out, err = output.Parse("gateway", nil)
	}
	if err != nil {
		return nil, err
	}

	baseConf := &base.Conf{}
	var executeRule *collectconfig.ExecuteRule
	if err := json.Unmarshal(task.Config.Content, baseConf); err == nil {
		executeRule = baseConf.ExecuteRule
	}

	if executeRule == nil {
		return nil, errors.New("executeRule is nil")
	}

	tags := make(map[string]string, len(baseConf.RefMetas))
	target := task.Target

	// TODO 列出所有 target type
	if target.IsTypePod() {
		for key, item := range baseConf.RefMetas {
			value := ""
			switch item.Name {
			case "app":
				value = target.GetApp()
			case "ip":
				value = target.GetIP()
			case "host":
				fallthrough
			case "hostname":
				value = target.GetHostname()
			default:
				value = target.Meta[value]
			}
			tags[key] = value
		}
	} else {
		for key, item := range baseConf.RefMetas {
			value := target.Meta[key]
			if value == "" && target.Type == collecttask.TargetLocalhost {
				switch item.Name {
				case "app":
					value = appconfig.StdAgentConfig.App
				case "ip":
					value = util.GetLocalIp()
				case "host":
					fallthrough
				case "hostname":
					value = util.GetHostname()
				}
			}
			tags[key] = value
		}
	}

	tenant := task.Target.GetTenant()
	to := &Output{
		Tenant: tenant,
		o:      out,
	}
	return NewPipeline(executeRule, task, i, to, tags)
}
