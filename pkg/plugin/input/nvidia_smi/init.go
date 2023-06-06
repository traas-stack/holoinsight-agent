/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package nvidia_smi

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/standard/providers"
	"os"
	"path/filepath"
)

// toolPath is the path of nvidia-smi
const toolPath = "/usr/bin/nvidia-smi"

// IsNvidiaEnabled checks if nvidia is enabled
func IsNvidiaEnabled() bool {
	switch appconfig.StdAgentConfig.Mode {
	case core.AgentModeDaemonset:
		if _, err := os.Stat(filepath.Join(core.GetHostfs(), toolPath)); err == nil {
			return true
		}
	case core.AgentModeSidecar:
		if _, err := os.Stat(toolPath); err == nil {
			return true
		}
	}
	return false
}

func init() {
	providers.RegisterInputProvider("gpu", func(task *collecttask.CollectTask) (api.Input, error) {
		return &NvidiaInput{}, nil
	})
}
