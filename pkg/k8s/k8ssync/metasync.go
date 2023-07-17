/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8ssync

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8sutils"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/listwatchext"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry"
	regpb "github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
)

const (
	defaultListWatchRestartInterval = 10 * time.Minute
	defaultEventMergeInterval       = 3 * time.Second
)

type (
	MetaSyncer interface {
		Start()
		Stop()
	}
	metaSyncer struct {
		rs               *registry.Service
		clientset        *kubernetes.Clientset
		namespaceChanged int32
		nodeChanged      int32
		podChanged       int32
	}
	convertFunc      func(obj interface{}) *Resource
	deleteStatusFunc func(obj interface{}) bool
	resourceFuncs    struct {
		convert convertFunc
		delete  deleteStatusFunc
	}
	resourceSyncer struct {
		ms         *metaSyncer
		getter     cache.Getter
		resource   string
		namespace  string
		objType    runtime.Object
		reportType string
		funcs      resourceFuncs
		verbose    bool
	}
)

func NewMetaSyncer(rs *registry.Service, clientset *kubernetes.Clientset) MetaSyncer {
	return &metaSyncer{
		rs:        rs,
		clientset: clientset,
	}
}

func newResourceSyncer(ms *metaSyncer, resource string, namespace string, objType runtime.Object, reportType string, funcs resourceFuncs) *resourceSyncer {
	return &resourceSyncer{
		ms:         ms,
		getter:     ms.clientset.CoreV1().RESTClient(),
		resource:   resource,
		namespace:  namespace,
		objType:    objType,
		reportType: reportType,
		funcs:      funcs,
	}
}

// 开始监听
func (rs *resourceSyncer) start(stopCh <-chan struct{}) {

	lw := listwatchext.NewListWatchFromClient(rs.getter, rs.resource, rs.namespace)

	em := listwatchext.NewEventMerger(defaultEventMergeInterval, func(merged map[string]watch.Event) {
		req := &DeltaSyncRequest{
			Apikey:    appconfig.StdAgentConfig.ApiKey,
			Type:      rs.reportType,
			Workspace: appconfig.StdAgentConfig.Workspace,
			Cluster:   appconfig.StdAgentConfig.Cluster,
		}

		for _, e := range merged {
			resource := rs.funcs.convert(e.Object)

			deleted := e.Type == watch.Deleted
			if !deleted && rs.funcs.delete != nil {
				deleted = rs.funcs.delete(e.Object)
			}
			if deleted {
				req.Del = append(req.Del, resource)
			} else {
				req.Add = append(req.Add, resource)
			}
		}

		if code, body, cost, err := rs.ms.syncToReg(req); err == nil {
			logger.Metaz("delta sync success",
				zap.Int("code", code),
				zap.String("body", body),
				zap.String("type", rs.reportType),
				zap.Int("add", len(req.Add)),
				zap.Int("del", len(req.Del)),
				zap.Duration("cost", cost))
			if rs.verbose {
				logger.Metaz("delta sync success", zap.Any("req", req))
			}
		} else {
			logger.Metaz("delta sync error",
				zap.Int("code", code),
				zap.String("body", body),
				zap.String("type", rs.reportType),
				zap.Int("add", len(req.Add)),
				zap.Int("del", len(req.Del)),
				zap.Duration("cost", cost),
				zap.Error(err))
			if rs.verbose {
				logger.Metaz("delta sync error", zap.Any("req", req))
			}
		}
	}, stopCh)
	helper := listwatchext.NewListWatchHelper(lw, listwatchext.ListWatchCallback{
		OnList: func(items []runtime.Object) {

			req := &FullSyncRequest{
				Apikey:    appconfig.StdAgentConfig.ApiKey,
				Type:      rs.reportType,
				Workspace: appconfig.StdAgentConfig.Workspace,
				Cluster:   appconfig.StdAgentConfig.Cluster,
			}

			for _, item := range items {
				if rs.funcs.delete != nil && rs.funcs.delete(item) {
					continue
				}
				req.Resources = append(req.Resources, rs.funcs.convert(item))
			}

			if code, body, cost, err := rs.ms.syncToReg(req); err == nil {
				logger.Metaz("full sync success",
					zap.Int("code", code),
					zap.String("body", body),
					zap.String("type", rs.reportType),
					zap.Int("size", len(items)),
					zap.Duration("cost", cost))
			} else {
				logger.Metaz("full sync error",
					zap.Int("code", code),
					zap.String("body", body),
					zap.String("type", rs.reportType),
					zap.Duration("cost", cost),
					zap.Int("size", len(items)),
					zap.Error(err))
			}
		},
		OnEvent: func(e watch.Event) {
			if rs.verbose {
				if ma, err := meta.Accessor(e.Object); err == nil {
					ns := ma.GetNamespace()
					name := ma.GetName()
					logger.Metaz("onEvent", zap.String("type", string(e.Type)), zap.String("ns", ns), zap.String("name", name))
				}
			}
			em.Add(e)
		},
	})

	go helper.Run(stopCh)
}

func (s *metaSyncer) Stop() {}

func (s *metaSyncer) Start() {
	// 步骤
	// 1. list, 然后立即与reg进行一次全量同步, 如果失败则退避重试
	// 2. watch, 攒批然后与reg进行增量同步, 遇到失败可重试(因为是幂等的), 如果重试无法成功则退出重新list&watch
	// 3. 兜底测策略, 每N(默认10)分钟强制重新list&watch一次
	go s.runLoop()
}

func (s *metaSyncer) runLoop() {
	start := time.NewTimer(0)
	defer start.Stop()

	stop := time.NewTimer(defaultListWatchRestartInterval)
	defer stop.Stop()

	var stopCh chan struct{}
	for {
		select {
		case <-start.C:
			stopCh = make(chan struct{})
			logger.Metaz("[listwatch] start")
			s.runOnce(stopCh)
			stop.Reset(defaultListWatchRestartInterval)
		case <-stop.C:
			close(stopCh)
			logger.Metaz("[listwatch] stop and restart after 1 s")
			start.Reset(time.Second)
		}
	}
}

func (s *metaSyncer) runOnce(stopCh <-chan struct{}) {
	{
		rs := newResourceSyncer(s, "nodes", v1.NamespaceAll, &v1.Node{}, "NODE", resourceFuncs{
			convert: func(obj interface{}) *Resource {
				return s.convertNodeToResource(obj.(*v1.Node))
			},
			delete: func(obj interface{}) bool {
				node := obj.(*v1.Node)
				return node.Status.Phase == v1.NodeTerminated
			},
		})
		rs.start(stopCh)
	}
	{
		rs := newResourceSyncer(s, string(v1.ResourcePods), v1.NamespaceAll, &v1.Pod{}, "POD", resourceFuncs{
			convert: func(obj interface{}) *Resource {
				return s.convertPodToResource(obj.(*v1.Pod))
			},
			delete: func(obj interface{}) bool {
				pod := obj.(*v1.Pod)
				return pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed
			},
		})
		rs.verbose = true
		rs.start(stopCh)
	}
}

func (s *metaSyncer) syncToReg(req interface{}) (int, string, time.Duration, error) {
	// 可以适当重试
	begin := time.Now()
	var err error
	switch x := req.(type) {
	case *FullSyncRequest:
		pbReq := &regpb.MetaSync_FullSyncRequest{
			Workspace: x.Workspace,
			Cluster:   x.Cluster,
			Type:      x.Type,
			Resource:  convertToPbResourceSlice(x.Resources),
		}
		err = s.rs.MetaFullSync(pbReq)
	case *DeltaSyncRequest:
		pbReq := &regpb.MetaSync_DeltaSyncRequest{
			Workspace: x.Workspace,
			Cluster:   x.Cluster,
			Type:      x.Type,
			Add:       convertToPbResourceSlice(x.Add),
			Del:       convertToPbResourceSlice(x.Del),
		}
		err = s.rs.MetaDeltaSync(pbReq)
	default:
		err = errors.New("unsupported sync type")
	}
	return 0, "", time.Now().Sub(begin), err
}

func (s *metaSyncer) convertNodeToResource(node *v1.Node) *Resource {
	ip := k8sutils.GetNodeIP(node)
	return &Resource{
		Name:        node.Name,
		Namespace:   "",
		Labels:      node.Labels,
		Annotations: node.Annotations,
		Ip:          ip,
		HostIP:      ip,
		Hostname:    k8smetaextractor.PodMetaServiceInstance.NodeHostname(node),
		Status:      string(node.Status.Phase),
	}
}

func (s *metaSyncer) convertPodToResource(pod *v1.Pod) *Resource {
	app := k8smetaextractor.DefaultPodMetaService.ExtractApp(pod)
	hostname := k8smetaextractor.PodMetaServiceInstance.ExtractHostname(pod)

	return &Resource{
		Name:        pod.Name,
		Namespace:   pod.Namespace,
		App:         app,
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		Ip:          pod.Status.PodIP,
		Hostname:    hostname,
		HostIP:      pod.Status.HostIP,
		Status:      string(pod.Status.Phase),
	}
}
