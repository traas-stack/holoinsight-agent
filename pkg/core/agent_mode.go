package core

import "errors"

type AgentMode string

const (
	AgentModeSidecar      AgentMode = "sidecar"
	AgentModeDaemonset    AgentMode = "daemonset"
	AgentModeClusteragent AgentMode = "clusteragent"
	AgentModeCentral      AgentMode = "central"
)

func ParseAgentMode(mode string) (AgentMode, error) {
	switch mode {
	case string(AgentModeSidecar):
		return AgentModeSidecar, nil
	case string(AgentModeDaemonset):
		return AgentModeDaemonset, nil
	case string(AgentModeClusteragent):
		return AgentModeClusteragent, nil
	case string(AgentModeCentral):
		return AgentModeCentral, nil
	}
	return "", errors.New("invalid mode " + mode)
}
