/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package bootstrap

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/containerdutils"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/dockerutils"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/impl/engine"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
)

func InitContainerdEngine() (cri.ContainerEngine, error) {
	client, versionResp, err := containerdutils.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	logger.Infoz("[bootstrap] [cri] init containerd client", zap.String("addr", client.Conn().Target()), zap.Any("version", versionResp))
	logger.Criz("[bootstrap] containerd client", zap.String("addr", client.Conn().Target()), zap.Any("version", versionResp))
	return &engine.ContainerdContainerEngine{Client: client}, nil
}

func InitDockerEngine() (cri.ContainerEngine, error) {
	docker, pingResp, err := dockerutils.NewClientFromEnv()
	if err != nil {
		logger.Errorz("[bootstrap] init docker client error", zap.Error(err))
		return nil, err
	}
	logger.Infoz("[bootstrap] init docker client", zap.String("host", docker.DaemonHost()), zap.Any("ping", pingResp))
	logger.Criz("[bootstrap] docker client", zap.String("host", docker.DaemonHost()), zap.Any("ping", pingResp))
	return &engine.DockerContainerEngine{Client: docker}, nil
}

func InitCollectTaskManager(rs *registry.Service, staticTasks []*collecttask.CollectTask) (*collecttask.Manager, error) {
	ctm, err := collecttask.NewManager(rs, agentmeta.GetAgentId())
	if err != nil {
		return nil, err
	}
	ctm.AddStaticTasks(staticTasks...)
	ctm.InitLoad()
	ioc.CollectTaskManager = ctm
	ctm.StartListen()
	ctm.AddHttpFuncs()
	App.AddStopComponent(ctm)
	return ctm, nil
}

func maybeInitDockerOOMManager() {
	i := ioc.Crii
	e := i.Engine()
	if x, ok := e.(*engine.DockerContainerEngine); ok {
		oomManager := engine.NewOOMManager(i, x.Client)
		oomManager.Start()
		App.AddStopComponent(oomManager)
	}
}

func InitK8sClientsetInCluster() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	config = metadata.ConfigFor(config)
	clientset, err := kubernetes.NewForConfig(config)
	if err == nil {
		ioc.K8sClientset = clientset
	}
	return clientset, err
}

func SetupConfigForK8s() {
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

func CreateContainerEngine() (cri.ContainerEngine, error) {
	switch appconfig.StdAgentConfig.K8s.Cri.Type {
	case "containerd":
		return InitContainerdEngine()
	case "docker":
		return InitDockerEngine()
	case "":
		engine, err := InitDockerEngine()
		if err != nil {
			engine, err = InitContainerdEngine()
			if err != nil {
				err = errors.New("fail to init container engine automatically")
			}
		}
		return engine, err
	default:
		return nil, errors.New("unsupported cri type: " + appconfig.StdAgentConfig.K8s.Cri.Type)
	}
}
