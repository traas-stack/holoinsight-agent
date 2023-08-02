/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package openmetric

import (
	"context"
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/impl/netproxy"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/oklog/run"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/scrape"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

type (
	// 我们将 prometheus 的服务端集成进来(采集能力)
	// 将我们的配置转成 prometheus 识别的配置
	// 配置 remote storage 写到 gateway
	// 本文件的大部分内容是 prometheus 初始化流程的简化版
	Manager struct {
		ctm *collecttask.Manager

		g         run.Group
		term      chan struct{}
		cancel    chan struct{}
		reloaders []reloader

		mergedConf *config.Config
		jobs       map[string]*config.ScrapeConfig
	}
	reloader struct {
		name     string
		reloader func(*config.Config) error
	}
	Logger struct {
	}
)

func (l *Logger) Log(keyvals ...interface{}) error {
	logger.Infof("[openmetric] [promehtues] [internal] %+v", keyvals)
	return nil
}

func NewManager(ctm *collecttask.Manager) *Manager {
	storage := &Storage{}
	promLogger := &Logger{}
	ctxScrape, cancelScrape := context.WithCancel(context.Background())

	discoveryManagerScrape := discovery.NewManager(ctxScrape, log.With(promLogger, "component", "discovery manager scrape"), discovery.Name("scrape"))

	scrapeManager := scrape.NewManager(log.With(promLogger, "component", "scrape manager"), storage)

	reloaders := []reloader{
		{
			// The Scrape and notifier managers need to reload before the Discovery manager as
			// they need to read the most updated config when receiving the new targets list.
			name:     "scrape",
			reloader: scrapeManager.ApplyConfig,
		}, {
			name: "scrape_sd",
			reloader: func(cfg *config.Config) error {
				logger.Infof("[openmetric] scrape_sd apply")
				c := make(map[string]discovery.Configs)
				for _, v := range cfg.ScrapeConfigs {
					c[v.JobName] = v.ServiceDiscoveryConfigs
				}
				return discoveryManagerScrape.ApplyConfig(c)
			},
		},
	}

	type closeOnce struct {
		C     chan struct{}
		once  sync.Once
		Close func()
	}
	reloadReady := &closeOnce{
		C: make(chan struct{}),
	}

	var g run.Group

	term := make(chan struct{}, 1)
	cancel := make(chan struct{})
	{
		g.Add(
			func() error {
				select {
				case <-term:
					level.Warn(promLogger).Log("msg", "Received SIGTERM, exiting gracefully...")
					if reloadReady.Close != nil {
						reloadReady.Close()
					}
				case <-cancel:
					if reloadReady.Close != nil {
						reloadReady.Close()
					}
				}
				return nil
			},
			func(err error) {
				close(cancel)
			},
		)
	}

	{
		// Scrape discovery manager.
		g.Add(
			func() error {
				err := discoveryManagerScrape.Run()
				level.Info(promLogger).Log("msg", "Scrape discovery manager stopped")
				return err
			},
			func(err error) {
				level.Info(promLogger).Log("msg", "Stopping scrape discovery manager...")
				cancelScrape()
			},
		)
	}

	{
		// Scrape manager.
		g.Add(
			func() error {
				// When the scrape manager receives a new targets list
				// it needs to read a valid config for each job.
				// It depends on the config being in sync with the discovery manager so
				// we wait until the config is fully loaded.
				// <-reloadReady.C
				logger.Infoz("[openmetric] scrapeManager start")
				err := scrapeManager.Run(discoveryManagerScrape.SyncCh())
				level.Info(promLogger).Log("msg", "Scrape manager stopped")
				logger.Infoz("[openmetric] scrapeManager stop", zap.Error(err))
				return err
			},
			func(err error) {
				// Scrape manager needs to be stopped before closing the local TSDB
				// so that it doesn't try to write samples to a closed storage.
				// We should also wait for rule manager to be fully stopped to ensure
				// we don't trigger any false positive alerts for rules using absent().
				level.Info(promLogger).Log("msg", "Stopping scrape manager...")
				scrapeManager.Stop()
			},
		)
	}

	return &Manager{
		ctm:       ctm,
		g:         g,
		cancel:    cancel,
		term:      term,
		reloaders: reloaders,
		mergedConf: &config.Config{
			GlobalConfig: config.GlobalConfig{
				ScrapeInterval: model.Duration(10 * time.Second),
				ScrapeTimeout:  model.Duration(3 * time.Second),
			},
		},
		jobs: make(map[string]*config.ScrapeConfig),
	}
}

func (m *Manager) Start() {
	logger.Infof("[openmetric] start")
	go m.g.Run()

	// 初始化
	all := m.ctm.GetAll()
	changed := false
	for _, t := range all {
		if !isPrometheusTask(t) {
			continue
		}
		scrapeConfig, err := convertToScrapeConfig(t)
		if err != nil {
			logger.Errorz("[openmetric] fail to parse task", zap.String("key", t.Key), zap.String("config", string(t.Config.Content)), zap.Error(err))
			continue
		}
		logger.Infoz("[openmetric] init", zap.Any("scrapeConfig", scrapeConfig))
		m.jobs[t.Key] = scrapeConfig
		changed = true
	}
	//
	//logger.Infoz("[openmetric] add kubernetes-pod")
	//
	//changed = true
	//// m.jobs["test"] = makeTestJob()
	//
	//m.jobs["kubernetes-pod"] = &config.ScrapeConfig{
	//	JobName:         "kubernetes-pod",
	//	HonorLabels:     true,
	//	HonorTimestamps: false,
	//	Params:          nil,
	//	ScrapeInterval:  model.Duration(15 * time.Second),
	//	ScrapeTimeout:   model.Duration(5 * time.Second),
	//	MetricsPath:     "/metrics",
	//	Scheme:          "http",
	//	SampleLimit:     0,
	//	TargetLimit:     0,
	//	ServiceDiscoveryConfigs: discovery.Configs{
	//		&MyK8sConfig{},
	//	},
	//	HTTPClientConfig: config2.HTTPClientConfig{
	//		TLSConfig: config2.TLSConfig{
	//			CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
	//		},
	//		BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
	//	},
	//	RelabelConfigs: []*relabel.Config{
	//		{
	//			SourceLabels: model.LabelNames{"__meta_kubernetes_pod_annotation_prometheus_io_scrape"},
	//			Action:       "keep",
	//			//TargetLabel:  "namespace",
	//			Separator:   ";",
	//			Regex:       relabel.MustNewRegexp("true"),
	//			Replacement: "$1",
	//		},
	//		{
	//			SourceLabels: model.LabelNames{"__meta_kubernetes_pod_annotation_prometheus_io_path"},
	//			Action:       "replace",
	//			TargetLabel:  "__metrics_path__",
	//			Separator:    ";",
	//			Regex:        relabel.MustNewRegexp("(.*)"),
	//			Replacement:  "$1",
	//		},
	//		{
	//			SourceLabels: model.LabelNames{"__address__", "__meta_kubernetes_pod_annotation_prometheus_io_port"},
	//			Action:       "replace",
	//			TargetLabel:  "__address__",
	//			Separator:    ";",
	//			Regex:        relabel.MustNewRegexp("([^:]+)(?::\\d+)?;(\\d+)"),
	//			Replacement:  "$1:$2",
	//		},
	//		{
	//			SourceLabels: model.LabelNames{"__meta_kubernetes_namespace"},
	//			Action:       "replace",
	//			TargetLabel:  "namespace",
	//			Separator:    ";",
	//			Regex:        relabel.MustNewRegexp("(.*)"),
	//			Replacement:  "$1",
	//		},
	//		{
	//			SourceLabels: model.LabelNames{"__meta_kubernetes_pod_name"},
	//			Action:       "replace",
	//			TargetLabel:  "pod",
	//			Separator:    ";",
	//			Regex:        relabel.MustNewRegexp("(.*)"),
	//			Replacement:  "$1",
	//		},
	//		{
	//			SourceLabels: model.LabelNames{"__meta_kubernetes_pod_container_name"},
	//			Action:       "replace",
	//			TargetLabel:  "container",
	//			Separator:    ";",
	//			Regex:        relabel.MustNewRegexp("(.*)"),
	//			Replacement:  "$1",
	//		},
	//		{
	//			SourceLabels: model.LabelNames{"__meta_kubernetes_pod_ip"},
	//			Action:       "replace",
	//			TargetLabel:  "ip",
	//			Separator:    ";",
	//			Regex:        relabel.MustNewRegexp("(.*)"),
	//			Replacement:  "$1",
	//		},
	//	},
	//	MetricRelabelConfigs: nil,
	//}

	if changed {
		jobs := m.jobs
		scrapeConfigs := make([]*config.ScrapeConfig, 0, len(jobs))
		for _, v := range jobs {
			scrapeConfigs = append(scrapeConfigs, v)
		}
		m.mergedConf.ScrapeConfigs = scrapeConfigs

		m.reload()
	}
	m.ctm.Listen(m)
}

func makeTestJob() *config.ScrapeConfig {
	var targets []model.LabelSet

	targets = append(targets, model.LabelSet{
		model.AddressLabel: model.LabelValue(fmt.Sprintf("%s:%s", "127.0.0.1", "9117")),
	})

	sc := &config.ScrapeConfig{
		JobName:         "test",
		HonorLabels:     true,
		HonorTimestamps: true,
		Params:          nil,
		ScrapeInterval:  model.Duration(15 * time.Second),
		ScrapeTimeout:   model.Duration(5 * time.Second),
		MetricsPath:     "/metrics",
		Scheme:          "http",
		SampleLimit:     0,
		TargetLimit:     0,
		ServiceDiscoveryConfigs: discovery.Configs{
			discovery.StaticConfig{
				{
					Targets: targets,
				},
			},
		},
		RelabelConfigs:       nil,
		MetricRelabelConfigs: nil,
	}
	addNetProxy(sc)

	return sc
}

func addNetProxy(sc *config.ScrapeConfig) {
	if !netproxy.NETPROXY_ENABLED {
		return
	}
	sc.HTTPClientConfig.ProxyURL.URL = netproxy.HttpProxyURL
}

// 判断是否为一个prometheus目录
func isPrometheusTask(t *collecttask.CollectTask) bool {
	return strings.HasSuffix(t.Config.Type, ".OpenmetricsScraperDTO")
}

func (m *Manager) OnUpdate(delta *collecttask.Delta) {
	jobs := make(map[string]*config.ScrapeConfig, len(m.jobs))
	for k, v := range m.jobs {
		jobs[k] = v
	}

	changed := false
	for _, t := range delta.Add {
		if !isPrometheusTask(t) {
			continue
		}
		changed = true
		scrapeConfig, err := convertToScrapeConfig(t)
		if err != nil {
			logger.Errorz("[openmetric] fail to parse task", zap.String("key", t.Key), zap.String("config", string(t.Config.Content)), zap.Error(err))
			continue
		}
		logger.Infoz("[openmetric] delta add", zap.Any("scrapeConfig", scrapeConfig))
		jobs[t.Key] = scrapeConfig
	}
	for _, t := range delta.Del {
		if !isPrometheusTask(t) {
			continue
		}
		logger.Infoz("[openmetric] delta del", zap.String("key", t.Key))
		delete(jobs, t.Key)
		changed = true
	}

	if changed {
		m.jobs = jobs
		scrapeConfigs := make([]*config.ScrapeConfig, 0, len(jobs))
		for _, v := range jobs {
			scrapeConfigs = append(scrapeConfigs, v)
		}
		m.mergedConf.ScrapeConfigs = scrapeConfigs
		m.reload()
	}
}

func (m *Manager) Stop() {
	m.term <- struct{}{}
}

func (m *Manager) StartListen() {
	m.ctm.Listen(m)
}

func (m *Manager) reload() {
	for _, rl := range m.reloaders {
		if err := rl.reloader(m.mergedConf); err != nil {
			logger.Errorz("[openmetric] apply error", zap.Error(err))
		}
	}
}
