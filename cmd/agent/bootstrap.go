/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package main

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/agent/server"
	bootstrap2 "github.com/traas-stack/holoinsight-agent/pkg/bootstrap"
	"github.com/traas-stack/holoinsight-agent/pkg/clusteragent"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/containerdutils"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/dockerutils"
	daemonsetmeta "github.com/traas-stack/holoinsight-agent/pkg/daemonset/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssync"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics"
	"github.com/traas-stack/holoinsight-agent/pkg/master"
	"github.com/traas-stack/holoinsight-agent/pkg/openmetric"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/bistream"
	pb2 "github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/agent"
	"github.com/traas-stack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	bizbistream "github.com/traas-stack/holoinsight-agent/pkg/bistream"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/all"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/output/all"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
)

type (
	stopComponent interface {
		Stop()
	}
	appStruct struct {
		stopComponents []stopComponent
	}
)

// 实例信息
var app = appStruct{}

func bootstrap() error {
	// 启动流程 begin
	begin := time.Now()

	// 加载应用基本配置
	// 初始化一些基础模块
	if err := appconfig.SetupAppConfig(); err != nil {
		return err
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

	if os.Getenv("DEBUG") == "true" {
		logger.DebugEnabled = true
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
	ioc.RegistryService = rs
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

	// 开启agent与registry之间的注册/心跳模块
	am := agent.NewManager(rs)
	am.Start()

	app.addStopComponent(am, rs)

	switch appconfig.StdAgentConfig.Mode {
	case core.AgentModeDaemonset:
		setupForK8s()

		if err := initK8sMetaManager(); err != nil {
			return err
		}

		if err := initCri(); err != nil {
			return err
		}

		// 5s system metrics
		//{
		//	c := k8ssync.NewPodSystemResourceCollector(m.MetaManager, "_5s", 5*time.Second)
		//	c.Start()
		//}

		// minute system metrics
		{
			if c, err := k8ssysmetrics.NewPodSystemResourceCollector("", time.Minute); err == nil {
				logger.Infof("[bootstrap] [k8s] use %s system metrics collector", c.Name())
				c.Start()
				app.addStopComponent(c)
			}
		}

		bootstrap2.TriggerDaemonsetHooks()

		ctm, err := initCollectTaskManager(rs)
		if err != nil {
			return err
		}

		pm := pipeline.NewManager(ctm)
		pm.Start()

		om := openmetric.NewManager(ctm)
		om.Start()

		bsm := bistream.NewManager(rs, bizbistream.GetBiStreamHandlerRegistry())
		bsm.Start()

		app.addStopComponent(pm, om, bsm)

		if appconfig.StdAgentConfig.Daemonagent.ClusterAgentEnabled {
			masterMaintainer := master.NewK8sNodeMasterMaintainer(ioc.K8smm)
			masterMaintainer.Register(&clusteragent.MasterComponent{})
			go masterMaintainer.Start()
			app.addStopComponent(masterMaintainer)
		}

	case core.AgentModeClusteragent:
		setupForK8s()

		if _, err := initK8sClientset(); err != nil {
			return err
		}

		k8sm := k8ssync.NewMetaSyncer(rs, ioc.K8sClientset)
		k8sm.Start()
		app.addStopComponent(k8sm)
	case core.AgentModeCentral:

		ctm, err := initCollectTaskManager(rs)
		if err != nil {
			return err
		}

		om := openmetric.NewManager(ctm)
		om.Start()

		pm := pipeline.NewManager(ctm)
		pm.Start()

		app.addStopComponent(om, pm)

	default:
	}

	logger.Infoz("[bootstrap] start success", zap.Int("pid", os.Getpid()), zap.Duration("cost", time.Now().Sub(begin)))
	go server.StartHTTPController()
	// bootstrap end

	return waitStop()
}

func setupForK8s() {
	i := &k8smetaextractor.PodMetaServiceInstance

	if s := appconfig.StdAgentConfig.K8s.Meta.AppRef; s != "" {
		logger.Infoz("[bootstrap] [k8s] [meta] override app extractor", zap.String("expr", s))
		f := k8smetaextractor.CreateLabelBasedPodMetaExtractFunc(s)
		i.App = k8smetaextractor.Composite(f, i.App)
	}

	if s := appconfig.StdAgentConfig.K8s.Meta.HostnameRef; s != "" {
		logger.Infoz("[bootstrap] [k8s] [meta] override hostname extractor", zap.String("expr", s))
		f := k8smetaextractor.CreateLabelBasedPodMetaExtractFunc(s)
		i.Hostname = k8smetaextractor.Composite(f, i.Hostname)
	}

	if s := appconfig.StdAgentConfig.K8s.Meta.NodeHostnameRef; s != "" {
		logger.Infoz("[bootstrap] [k8s] [meta] override nodeHostname extractor", zap.String("expr", s))
		f := k8smetaextractor.CreateLabelBasedNodeMetaExtractFunc(s)
		i.NodeHostname = k8smetaextractor.NodeComposite(f, i.NodeHostname)
	}

	if s := appconfig.StdAgentConfig.K8s.Meta.SidecarCheck; s != "" {
		logger.Infoz("[bootstrap] [k8s] [meta] override sidecar check", zap.String("expr", s))
		i.Sidecar = k8smetaextractor.NewDefaultSidecarCheckHook(s)
	}

	if len(appconfig.StdAgentConfig.K8s.Meta.Sandbox.Labels) > 0 {
		logger.Infoz("[bootstrap] [k8s] [meta] override sandbox check", zap.Any("sandbox", appconfig.StdAgentConfig.K8s.Meta.Sandbox))
		i.Sandbox = k8smetaextractor.CreateLabelBasedSandboxCheckHook(appconfig.StdAgentConfig.K8s.Meta.Sandbox.Labels)
	}

}

func initK8sClientset() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err == nil {
		ioc.K8sClientset = clientset
	}
	return clientset, err
}

func initK8sMetaManager() error {
	clientset, err := initK8sClientset()
	if err != nil {
		return err
	}

	k8smm := k8smeta.NewManager(clientset)
	k8smm.Start()
	ioc.K8smm = k8smm
	app.addStopComponent(k8smm)

	return nil
}

func waitStop() (ret error) {
	c := make(chan os.Signal, 16)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	signal.Stop(c)
	logger.Infoz("[agent] receive stop signal", zap.String("signal", sig.String()), zap.Int("components", len(app.stopComponents)))

	begin0 := time.Now()
	for i := len(app.stopComponents) - 1; i >= 0; i-- {
		begin := time.Now()
		component := app.stopComponents[i]

		logger.Infoz("[agent] try to stop component", //
			zap.Int("index", i),                        //
			zap.Any("type", reflect.TypeOf(component)), //
		) //

		component.Stop()
		cost := time.Now().Sub(begin)
		logger.Infoz("[agent] stop component", //
			zap.Any("type", reflect.TypeOf(component)), //
			zap.Duration("cost", cost))                 //
	}
	cost := time.Now().Sub(begin0)
	logger.Infoz("[agent] stop done", zap.Duration("cost", cost))
	return nil
}

func initCri() error {
	var engine cri.ContainerEngine
	var err error
	switch appconfig.StdAgentConfig.K8s.Cri.Type {
	case "containerd":
		engine, err = initContainerdEngine()
	case "docker":
		engine, err = initDockerEngine()
	case "":
		engine, err = initDockerEngine()
		if err != nil {
			engine, err = initContainerdEngine()
			if err != nil {
				err = errors.New("fail to init cri automatically")
			}
		}
	default:
		return errors.New("unsupported cri type: " + appconfig.StdAgentConfig.K8s.Cri.Type)
	}
	if err != nil {
		return err
	}

	if err := engine.Init(); err != nil {
		logger.Errorz("[bootstrap] container engine init error", zap.String("engine", engine.Type()), zap.Error(err))
		return err
	}

	i := daemonsetmeta.New(ioc.K8smm, engine)
	ioc.Crii = i
	if err := i.Start(); err != nil {
		return err
	}
	app.addStopComponent(i)
	maybeInitDockerOOMManager()
	return nil
}

func maybeInitDockerOOMManager() {
	i := ioc.Crii
	engine := i.Engine()
	if x, ok := engine.(*daemonsetmeta.DockerContainerEngine); ok {
		oomManager := daemonsetmeta.NewOOMManager(i, x.Client)
		oomManager.Start()
		app.addStopComponent(oomManager)
	}
}

func initContainerdEngine() (cri.ContainerEngine, error) {
	client, versionResp, err := containerdutils.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	logger.Infoz("[bootstrap] [cri] init containerd client", zap.String("addr", client.Conn().Target()), zap.Any("version", versionResp))
	logger.Criz("[bootstrap] containerd client", zap.String("addr", client.Conn().Target()), zap.Any("version", versionResp))
	return &daemonsetmeta.ContainerdContainerEngine{Client: client}, nil
}
func initDockerEngine() (cri.ContainerEngine, error) {
	docker, pingResp, err := dockerutils.NewClientFromEnv()
	if err != nil {
		logger.Errorz("[bootstrap] init docker client error", zap.Error(err))
		return nil, err
	}
	logger.Infoz("[bootstrap] init docker client", zap.String("host", docker.DaemonHost()), zap.Any("ping", pingResp))
	logger.Criz("[bootstrap] docker client", zap.String("host", docker.DaemonHost()), zap.Any("ping", pingResp))
	return &daemonsetmeta.DockerContainerEngine{Client: docker}, nil
}

func initCollectTaskManager(rs *registry.Service) (*collecttask.Manager, error) {
	ctm, err := collecttask.NewManager(rs, agentmeta.GetAgentId())
	if err != nil {
		return nil, err
	}
	ctm.InitLoad()
	ioc.CollectTaskManager = ctm
	ctm.StartListen()
	ctm.AddHttpFuncs()
	app.addStopComponent(ctm)
	return ctm, nil
}

func (app *appStruct) addStopComponent(components ...stopComponent) {
	app.stopComponents = append(app.stopComponents, components...)
}
