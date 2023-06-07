/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package agent

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/nodehostname"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
)

const (
	// 每 1min 1次心跳
	heartbeatInterval = 1 * time.Minute
	// 每5次心跳转成一个注册
	registerInterval = 5
	firstRpcTimeout  = time.Second
	rpcTimeout       = 3 * time.Second
)

type (
	Manager struct {
		rs                *registry.Service
		stop              chan struct{}
		reportInfoDisable bool
		// 是否曾经注册成功过
		registerSucceed bool
		mutex           sync.Mutex
	}
)

func NewManager(rs *registry.Service) *Manager {
	return &Manager{
		rs:   rs,
		stop: make(chan struct{}, 1),
	}
}

func (m *Manager) Start() {
	// 连立即注册一次, 无论失败还是成功
	m.registerAgent(firstRpcTimeout)
	go m.loop()
}

func (m *Manager) runInLock(f func()) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	f()
}

func (m *Manager) isStopped() bool {
	select {
	case <-m.stop:
		return true
	default:
		return false
	}
}

func (m *Manager) loop() {
	heartbeatTimer := time.NewTimer(0)
	defer heartbeatTimer.Stop()

	//reportRunInfoTimer := time.NewTicker(time.Minute)
	//defer reportRunInfoTimer.Stop()

	refreshControlConfigTimer := time.NewTicker(defaultRunInfoReportDuration)
	defer refreshControlConfigTimer.Stop()

	count := registerInterval
	for {
		select {
		case <-heartbeatTimer.C:
			m.runInLock(func() {
				if m.isStopped() {
					return
				}
				count++
				// 如果不曾注册成功, 那就总是 register
				// TODO register 适当退避
				if !m.registerSucceed || count >= registerInterval {
					count = 0
					m.registerAgent(rpcTimeout)
				} else {
					m.sendHeartbeat()
				}
				heartbeatTimer.Reset(heartbeatInterval)
			})
		case <-refreshControlConfigTimer.C:
			m.runInLock(func() {
				if m.isStopped() {
					return
				}
				m.refreshAgentControlConfigs()
			})
		case <-m.stop:
			return
		}
	}
}

func (m *Manager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	close(m.stop)
}

func (m *Manager) registerAgent(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// TODO 这里耦合k8s
	var k8sinfo *pb.AgentK8SInfo
	if appconfig.IsK8sMode() {
		k8sinfo = &pb.AgentK8SInfo{
			HostIp:    os.Getenv("NODE_IP"),
			Namespace: os.Getenv("POD_NAMESPACE"),
			Pod:       os.Getenv("POD_NAME"),
		}
		if appconfig.StdAgentConfig.Mode == core.AgentModeDaemonset {
			k8sinfo.NodeHostname = nodehostname.NodeHostname()
		}
	}
	labels := map[string]string{}
	if appconfig.StdAgentConfig.Mode == core.AgentModeCentral {
		labels["central.name"] = appconfig.StdAgentConfig.Central.Name
	}
	resp, err := m.rs.RegisterAgent(ctx, registry.AgentInfo{
		AgentId:      agentmeta.GetAgentId(),
		AgentVersion: appconfig.StdAgentConfig.Version,
		Hostname:     util.GetHostname(),
		Ip:           util.GetLocalIp(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		App:          appconfig.StdAgentConfig.App,
		Labels:       labels,
		Mode:         string(appconfig.StdAgentConfig.Mode),
		K8s:          k8sinfo,
		Workspace:    appconfig.StdAgentConfig.Workspace,
	})
	if err != nil {
		logger.Errorz("[am] register error", zap.Stringer("resp", resp), zap.Error(err))
		m.registerSucceed = false
	} else {
		if resp.Header != nil && resp.Header.Code == 0 {
			m.registerSucceed = true
		} else {
			m.registerSucceed = false
		}
		logger.Infoz("[am] register", zap.Stringer("resp", resp))
	}
}

func (m *Manager) sendHeartbeat() {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp, err := m.rs.SendHeartBeat(ctx, agentmeta.GetAgentId())
	if err != nil {
		m.registerSucceed = false
		logger.Errorz("[am] heartbeat error", zap.Stringer("resp", resp), zap.Error(err))
	} else {
		logger.Infoz("[am] heartbeat", zap.Stringer("resp", resp))
	}
}

func (m *Manager) refreshAgentControlConfigs() {
	resp := m.rs.GetLastControlConfigs()
	if resp != nil {
		logger.Infoz("[am] get control configs", zap.Any("resp", resp))
		m.reportInfoDisable = resp.GetDebugConfig().GetReportDisable()
	}
}
