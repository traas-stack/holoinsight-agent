/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package main

import (
	"github.com/traas-stack/holoinsight-agent/pkg/agent/server"
	bootstrap2 "github.com/traas-stack/holoinsight-agent/pkg/bootstrap"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/dockerutils"
	daemonsetmeta "github.com/traas-stack/holoinsight-agent/pkg/daemonset/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssync"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics"
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
	"github.com/traas-stack/holoinsight-agent/pkg/openmetric"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/all"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/output/all"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
)

type (
	stopComponent interface {
		Stop()
	}
)

// 实例信息
var app = struct {
	stopComponents []stopComponent
	stop           chan struct{}
	stopped        chan error
	started        bool
}{
	stopped: make(chan error, 1),
}

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
		os.Exit(1)
	}

	if os.Getenv("DEBUG") == "true" {
		logger.DebugEnabled = true
	}

	{
		wd, err := os.Getwd()
		if err != nil {
			logger.Errorz("[bootstrap] get working directory error", zap.Error(err))
		} else {
			logger.Infoz("[bootstrap] working directory", zap.String("wd", wd))
		}
	}

	stat.SetupStat()

	// setup agent id
	err := agentmeta.SetupAgentId()
	if err != nil {
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

	// 初始化采集配置同步模块
	ctm, err := collecttask.NewManager(rs, agentmeta.GetAgentId())
	if err != nil {
		am.Stop()
		rs.Stop()
		return err
	}
	// 从本地加载采集配置
	// 立即从 Registry 同步一次配置 (拖慢启动速度)
	ctm.InitLoad()

	//// 启动 pipeline manager
	//pm := pipeline.NewManager(ctm)
	//pm.Start()

	// 启动监听
	ctm.StartListen()
	ctm.AddHttpFuncs()
	ioc.CollectTaskManager = ctm

	// 双向流
	bsm := bistream.NewManager(rs, bizbistream.GetBiStreamHandlerRegistry())
	bsm.Start()

	om := openmetric.NewManager(ctm)
	om.Start()

	app.stopComponents = append(app.stopComponents, am, rs, ctm, bsm)

	switch appconfig.StdAgentConfig.Mode {
	case core.AgentModeDaemonset:
		setupForK8s()

		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}

		k8smm := k8smeta.NewManager(clientset)
		k8smm.Start()
		ioc.K8smm = k8smm
		app.stopComponents = append(app.stopComponents, k8smm)

		sock, sockOk := dockerutils.DetectSock()
		if !sockOk {
			panic("no docker sock")
		}
		host := "unix://" + sock
		docker, pingResp, err := dockerutils.NewDockerClient(host)
		if err != nil {
			logger.Errorz("[bootstrap] init docker client error", zap.Error(err))
			time.Sleep(time.Second)
			panic(err)
		}
		logger.Infoz("[bootstrap] init docker client", zap.String("host", host), zap.Any("ping", pingResp))
		logger.Dockerz("[init] docker client", zap.String("host", host), zap.Any("ping", pingResp))

		dm := daemonsetmeta.New(rs, k8smm, docker)
		ioc.Crii = dm

		// 5秒级系统指标任务采集
		//{
		//	c := k8ssync.NewPodSystemResourceCollector(m.MetaManager, "_5s", 5*time.Second)
		//	c.Start()
		//}
		// 分钟级系统指标任务采集

		{
			if c, err := k8ssysmetrics.NewPodSystemResourceCollector("", time.Minute); err == nil {
				logger.Infof("[bootstrap] [k8s] use %s system metrics collector", c.Name())
				c.Start()
				app.stopComponents = append(app.stopComponents, c)
			}
		}

		bootstrap2.TriggerDaemonsetHooks()

	case core.AgentModeClusteragent:
		setupForK8s()

		// TODO 重复代码太多
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}

		k8smm := k8smeta.NewManager(clientset)
		k8smm.Start()
		ioc.K8smm = k8smm
		app.stopComponents = append(app.stopComponents, k8smm)

		k8sm := k8ssync.NewMetaSyncer(rs, k8smm)
		k8sm.Start()
		app.stopComponents = append(app.stopComponents, k8sm)
	case core.AgentModeCentral:
	default:
	}

	pm := pipeline.NewManager(ctm)
	pm.Start()
	app.stopComponents = append(app.stopComponents, pm)

	// 等待关闭
	// 如果是以交互方式运行 则监听 Ctrl+C
	logger.Infoz("[bootstrap] start success", zap.Int("pid", os.Getpid()), zap.Duration("cost", time.Now().Sub(begin)))
	app.started = true
	go server.StartHTTPController()

	// 启动流程 end

	if appconfig.IsInteractive() {
		// 如果处于交互式模式则监听退出信号
		c := make(chan os.Signal, 16)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		sig := <-c
		signal.Stop(c)
		logger.Infof("[agent] receive %s, stop agent", sig.String())
	} else {
		// 否则等待我们自己内部的一个信号
		// 另外一个地方会去操作这个信号
		app.stop = make(chan struct{})
		<-app.stop
	}

	return stop0()
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

func Stop() error {
	// 没有启动成功 那就不用stop
	if !app.started {
		return nil
	}

	// 如果处于非交互式模式, 则关闭这个chan, 触发程序停止
	if !appconfig.IsInteractive() {
		close(app.stop)
	}
	// 等待停止结果
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case err := <-app.stopped:
		return err
	case <-timer.C:
		// timeout
		return nil
	}
}

func stop0() (ret error) {
	defer func() {
		app.stopped <- ret
	}()

	for i := len(app.stopComponents) - 1; i >= 0; i-- {
		begin := time.Now()
		component := app.stopComponents[i]

		logger.Infoz("[agent] try to stop component", //
			zap.Any("type", reflect.TypeOf(component)), //
		) //

		component.Stop()
		end := time.Now()
		logger.Infoz("[agent] stop component", //
			zap.Any("type", reflect.TypeOf(component)), //
			zap.Duration("cost", end.Sub(begin)))       //
	}

	return nil
}
