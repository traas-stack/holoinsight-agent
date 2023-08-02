/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package bootstrap

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/agent"
	"github.com/traas-stack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/traas-stack/holoinsight-agent/pkg/agent/server"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	bizbistream "github.com/traas-stack/holoinsight-agent/pkg/bistream"
	"github.com/traas-stack/holoinsight-agent/pkg/clusteragent"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/impl"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssync"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/resources"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/master"
	"github.com/traas-stack/holoinsight-agent/pkg/openmetric"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/bistream"
	pb2 "github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/transfer/manager"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"
)

type (
	Runnable   func()
	Customizer func(obj interface{}) (interface{}, error)

	StopComponent interface {
		Stop()
	}

	AgentBootstrap struct {
		StopHooks           []Runnable
		Hooks               []Runnable
		Customizers         map[string][]Customizer
		StaticTasks         []*collecttask.CollectTask
		stopComponents      []StopComponent
		LSM                 *logstream.Manager
		PM                  *pipeline.Manager
		TM                  *manager.TransferManager
		httpServerComponent *server.HttpServerComponent
		AM                  *agent.Manager
	}
)

var (
	App = AgentBootstrap{
		Customizers: make(map[string][]Customizer),
	}
)

func (b *AgentBootstrap) AddCustomizer(name string, customizer Customizer) {
	b.Customizers[name] = append(b.Customizers[name], customizer)
}

func (b *AgentBootstrap) callCustomizers(name string, obj interface{}) interface{} {
	for _, customizer := range b.Customizers[name] {
		obj2, err := customizer(obj)
		if err != nil {
			panic(err)
		}
		if obj != nil && obj2 == nil {
			panic("customizer returns nil")
		}
		obj = obj2
	}
	return obj
}

func (b *AgentBootstrap) AddStopComponents(components ...StopComponent) {
	b.stopComponents = append(b.stopComponents, components...)
}

func (b *AgentBootstrap) Bootstrap() error {
	resources.SetupCpuAndMemoryLimit()

	begin := time.Now()

	// setup basic config
	if err := appconfig.SetupAppConfig(); err != nil {
		return err
	}

	if os.Getenv("DEBUG") == "true" {
		logger.DebugEnabled = true
	}

	if appconfig.StdAgentConfig.Mode == core.AgentModeDaemonset {
		manager.MaybePrepareTransfer()
	}

	// setup logger
	logger.SetupZapLogger()

	logger.Infoz("[bootstrap] config", zap.Any("config", appconfig.StdAgentConfig))

	if s := os.Getenv("POD_IP"); s != "" {
		util.SetLocalIp(s)
	}

	logger.Infoz("[bootstrap] network", zap.String("ip", util.GetLocalIp()), zap.String("hostname", util.GetHostname()))
	if util.GetLocalIp() == "" {
		logger.Errorz("[bootstrap] invalid ip")
		return errors.New("[bootstrap] invalid ip")
	}

	if wd, err := os.Getwd(); err != nil {
		logger.Errorz("[bootstrap] get working directory error", zap.Error(err))
		return err
	} else {
		logger.Infoz("[bootstrap] working directory", zap.String("wd", wd))
	}

	stat.SetupStat()

	// setup agent id
	if err := agentmeta.SetupAgentId(); err != nil {
		logger.Errorz("[bootstrap] setup agent id error", zap.Error(err))
		return err
	}
	logger.Infoz("[bootstrap] load agent id", zap.String("agentId", agentmeta.GetAgentId()))

	if err := b.setupRegistryService(); err != nil {
		return err
	}

	if err := b.setupAgentManager(); err != nil {
		return err
	}

	b.httpServerComponent = server.NewHttpServerComponent()
	b.AddStopComponents(b.httpServerComponent)

	switch appconfig.StdAgentConfig.Mode {
	case core.AgentModeDaemonset:
		if err := b.setupDaemonAgent(); err != nil {
			return err
		}

		go b.TM.ListenTransfer()

	case core.AgentModeClusteragent:
		if err := b.setupClusterAgent(); err != nil {
			return err
		}

	case core.AgentModeCentral:
		if err := b.setupCentralAgent(); err != nil {
			return err
		}
	case core.AgentModeSidecar:
		if err := b.setupSidecarAgent(); err != nil {
			return err
		}
	}

	logger.Infoz("[bootstrap] bootstrap success", zap.Int("pid", os.Getpid()), zap.Duration("cost", time.Now().Sub(begin)))
	b.httpServerComponent.Start()

	b.waitStop()
	return b.onStop()
}

func (b *AgentBootstrap) setupRegistryService() error {
	b.callCustomizers("registryService-setup-begin", nil)

	regConfig := appconfig.StdAgentConfig.Registry
	rs, err := registry.New(registry.Config{
		Addr:         regConfig.Addr,
		Secure:       regConfig.Secure,
		ServerName:   regConfig.ServiceName,
		CaCertBase64: regConfig.CaCert,
		Apikey:       appconfig.StdAgentConfig.ApiKey,
		AgentId:      agentmeta.GetAgentId(),
		Workspace:    appconfig.StdAgentConfig.Workspace,
	})
	if err != nil {
		return err
	}
	rs = b.callCustomizers("registryService", rs).(*registry.Service)
	ioc.RegistryService = rs
	b.AddStopComponents(rs)

	rs.ReportEventAsync(&pb2.ReportEventRequest_Event{
		BornTimestamp: time.Now().UnixMilli(),
		EventType:     "DIGEST",
		PayloadType:   "agent_bootstrap",
		Tags: map[string]string{
			"agent": agentmeta.GetAgentId(),
			"ip":    util.GetLocalIp(),
		},
		Json: util.ToJsonString(map[string]interface{}{
			"config": appconfig.StdAgentConfig,
		}),
	})

	b.callCustomizers("registryService-setup-end", nil)
	return nil
}

func (b *AgentBootstrap) setupAgentManager() error {
	b.callCustomizers("agentManager-setup-begin", nil)

	// setup agent manager
	am := agent.NewManager(ioc.RegistryService)
	am = b.callCustomizers("agentManager", am).(*agent.Manager)
	am.Start()
	b.AM = am
	b.AddStopComponents(am)

	b.callCustomizers("agentManager-setup-end", nil)
	return nil
}

func (b *AgentBootstrap) waitStop() {
	c := make(chan os.Signal, 16)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	signal.Stop(c)
	logger.Infoz("[agent] receive stop signal", zap.String("signal", sig.String()))
}

func (b *AgentBootstrap) callStopComponents() {
	for i := len(App.stopComponents) - 1; i >= 0; i-- {
		begin := time.Now()
		component := App.stopComponents[i]

		logger.Infoz("[agent] try to stop component", //
			zap.Int("index", i),                        //
			zap.Any("type", reflect.TypeOf(component)), //
		) //

		component.Stop()
		cost := time.Now().Sub(begin)
		logger.Infoz("[agent] stop component", //
			zap.Any("type", reflect.TypeOf(component)), //
			zap.Duration("cost", cost)) //
	}
}

func (b *AgentBootstrap) onStop() error {
	if b.TM != nil {
		b.TM.StopSaveStateToFile()
	}
	begin0 := time.Now()
	b.callStopComponents()
	b.callCustomizers("stop", nil)
	cost := time.Now().Sub(begin0)
	logger.Infoz("[agent] stop done", zap.Duration("cost", cost))
	return nil
}

func (b *AgentBootstrap) setupClusterAgent() error {
	SetupConfigForK8s()
	b.callCustomizers("clusteragent-setup-begin", nil)

	if _, err := InitK8sClientsetInCluster(); err != nil {
		return err
	}

	k8sm := k8ssync.NewMetaSyncer(ioc.RegistryService, ioc.K8sClientset)
	k8sm.Start()
	App.AddStopComponents(k8sm)

	b.callCustomizers("clusteragent-setup-end", nil)
	return nil
}

func (b *AgentBootstrap) setupCentralAgent() error {
	b.callCustomizers("centralagent-setup-begin", nil)
	ctm, err := InitCollectTaskManager(ioc.RegistryService, nil)
	if err != nil {
		return err
	}

	om := openmetric.NewManager(ctm)
	om.Start()

	lsm := logstream.NewManager()
	b.LSM = lsm

	lsm.Start()
	App.AddStopComponents(lsm)

	pm := pipeline.NewManager(ctm, lsm)
	pm.Start()
	App.AddStopComponents(om, pm)
	b.callCustomizers("centralagent-setup-end", nil)
	return nil
}

func (b *AgentBootstrap) setupDaemonAgent() error {
	SetupConfigForK8s()
	b.callCustomizers("daemonagent-setup-begin", nil)

	if _, err := InitK8sClientsetInCluster(); err != nil {
		return err
	}

	if err := b.setupCRI(); err != nil {
		return err
	}

	// system metrics
	{
		collector := k8ssysmetrics.GetNewPodSystemResourceCollector()
		b.StaticTasks = append(b.StaticTasks, &collecttask.CollectTask{
			Key: "syscollector",
			Config: &collecttask.CollectConfig{
				Key:  "syscollector",
				Type: "syscollector_" + collector,
			},
			Target: &collecttask.CollectTarget{
				Type: collecttask.TargetNone,
			},
		})
		logger.Infof("[bootstrap] [k8s] use %s system metrics collector", collector)
	}

	ctm, err := InitCollectTaskManager(ioc.RegistryService, b.StaticTasks)
	if err != nil {
		return err
	}

	lsm := logstream.NewManager()
	b.LSM = lsm

	lsm.Start()
	App.AddStopComponents(lsm)

	pm := pipeline.NewManager(ctm, lsm)
	b.PM = pm
	pm.LoadAll()

	bsm := bistream.NewManager(ioc.RegistryService, bizbistream.GetBiStreamHandlerRegistry())

	b.TM = manager.NewTransferManager(b.PM, b.LSM)
	b.TM.AddStopComponents(b.httpServerComponent, ctm, bsm, b.AM)
	if err := b.TM.Transfer(); err != nil {
		logger.Errorz("[transfer] error", zap.Error(err))
	}

	pm.Start()

	om := openmetric.NewManager(ctm)
	om.Start()

	bsm.Start()

	App.AddStopComponents(pm, om, bsm)

	masterMaintainer := master.NewK8sNodeMasterMaintainer(ioc.Crii, ioc.K8sClientset)
	masterMaintainer.Register(&clusteragent.MasterComponent{})
	go masterMaintainer.Start()
	App.AddStopComponents(masterMaintainer)

	b.callCustomizers("daemonagent-setup-end", nil)

	return nil
}

func (b *AgentBootstrap) setupSidecarAgent() error {
	b.callCustomizers("sidecaragent-setup-begin", nil)

	ctm, err := InitCollectTaskManager(ioc.RegistryService, nil)
	if err != nil {
		return err
	}

	lsm := logstream.NewManager()
	b.LSM = lsm

	lsm.Start()
	App.AddStopComponents(lsm)

	pm := pipeline.NewManager(ctm, lsm)
	pm.Start()

	bsm := bistream.NewManager(ioc.RegistryService, bizbistream.GetBiStreamHandlerRegistry())
	bsm.Start()

	App.AddStopComponents(pm, bsm)

	b.callCustomizers("sidecaragent-setup-end", nil)

	return nil
}

func (b *AgentBootstrap) setupCRI() error {
	b.callCustomizers("cri-setup-begin", nil)

	engine, err := CreateContainerEngine()
	if err != nil {
		return err
	}
	if err := engine.Init(); err != nil {
		logger.Errorz("[bootstrap] container engine init error", zap.String("engine", engine.Type()), zap.Error(err))
		return err
	}

	engine = b.callCustomizers("containerEngine-setup", engine).(cri.ContainerEngine)

	i := impl.NewDefaultCri(ioc.K8sClientset, engine)
	ioc.Crii = i
	if err := i.Start(); err != nil {
		return err
	}
	if b.TM != nil {
		b.TM.AddStopComponents(i)
	}
	App.AddStopComponents(i)
	maybeInitDockerOOMManager()
	b.callCustomizers("cri-setup-end", nil)
	return nil
}
