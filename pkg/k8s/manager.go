/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8s

//
//import (
//	"github.com/traas-stack/holoinsight-agent/internal/registry"
//	"github.com/traas-stack/holoinsight-agent/pkg/core"
//	daemonsetmeta "github.com/traas-stack/holoinsight-agent/pkg/daemonset/meta"
//	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta"
//	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssync"
//	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics"
//	"github.com/traas-stack/holoinsight-agent/pkg/logger"
//	"time"
//)
//
//type (
//	Manager struct {
//		rs          *registry.Service
//		agentMode   core.AgentMode
//		MetaManager *k8smeta.Manager
//		dm          *daemonsetmeta.DockerLocalMetaImpl
//	}
//)
//
//// TODO 废弃
//func NewManager(rs *registry.Service, k8smm *k8smeta.Manager, dm *daemonsetmeta.DockerLocalMetaImpl, agentMode core.AgentMode) *Manager {
//	return &Manager{
//		rs:          rs,
//		MetaManager: k8smm,
//		dm:          dm,
//		agentMode:   agentMode,
//	}
//}
//
//func (m *Manager) Start() error {
//	switch m.agentMode {
//	case core.AgentModeClusteragent:
//		k8ssync.NewMetaSyncer(m.MetaManager, m.rs).Start()
//	case core.AgentModeDaemonset:
//		// 5秒级系统指标任务采集
//		//{
//		//	c := k8ssync.NewPodSystemResourceCollector(m.MetaManager, "_5s", 5*time.Second)
//		//	c.Start()
//		//}
//		// 分钟级系统指标任务采集
//		{
//			if c, err := k8ssysmetrics.NewPodSystemResourceCollector(m.MetaManager, "", time.Minute); err == nil {
//				logger.Infof("[bootstrap] [k8s] use %s system metrics collector", c.Name())
//				c.Start()
//			}
//		}
//	}
//
//	return nil
//}
//
//func (m *Manager) Stop() {
//}
