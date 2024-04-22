/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package thread

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/standard/providers"
)

func init() {
	providers.RegisterInputProvider("countthreadtask", func(task *collecttask.CollectTask) (api.Input, error) {
		return &input{
			task: task,
		}, nil
	})
}
