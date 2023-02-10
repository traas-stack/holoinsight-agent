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
		stopped           chan struct{}
		reportInfoDisable bool
		// 是否曾经注册成功过
		registerSucceed bool
	}
)

func NewManager(rs *registry.Service) *Manager {
	return &Manager{
		rs:      rs,
		stop:    make(chan struct{}, 1),
		stopped: make(chan struct{}),
	}
}

func (m *Manager) Start() {
	// 连立即注册一次, 无论失败还是成功
	m.registerAgent(firstRpcTimeout)
	go m.loop()
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
		//case <-reportRunInfoTimer.C:
		//	m.reportRunInfo()
		case <-refreshControlConfigTimer.C:
			m.refreshAgentControlConfigs()
		case <-m.stop:
			m.stopped <- struct{}{}
			return
		}
	}
}

func (m *Manager) Stop() {
	m.stop <- struct{}{}
	close(m.stop)

	<-m.stopped
}

func (m *Manager) registerAgent(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// TODO 这里耦合k8s
	var k8sinfo *pb.AgentK8SInfo
	if appconfig.IsK8sMode() {
		k8sinfo = &pb.AgentK8SInfo{
			HostIp:    os.Getenv("HOST_IP"),
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
	} else {
		if resp.Header != nil && resp.Header.Code == 0 {
			m.registerSucceed = true
		}
		logger.Infoz("[am] register", zap.Stringer("resp", resp))
	}
}

func (m *Manager) sendHeartbeat() {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp, err := m.rs.SendHeartBeat(ctx, agentmeta.GetAgentId())
	if err != nil {
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
