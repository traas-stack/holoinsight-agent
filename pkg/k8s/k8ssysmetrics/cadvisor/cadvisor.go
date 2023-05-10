/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cadvisor

import (
	"context"
	"fmt"
	cadvisorclient "github.com/google/cadvisor/client"
	cv1 "github.com/google/cadvisor/info/v1"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8slabels"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8ssysmetrics/common"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/traas-stack/holoinsight-agent/pkg/util/trigger"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// cadvisor 是我们自己部署的 非主机网络模式 端口不会冲突
	cadvisorPort = 8080
)

type (
	cadvisorSysCollector struct {
		k8smm     *k8smeta.Manager
		crii      cri.Interface
		agentMode core.AgentMode
		suffix    string
		cache1    *sync.Map
		cache2    *sync.Map
		interval  time.Duration
		stopSig   *util.StopSignal
	}
	// pod cgroup 级别的labels上是没有相关标签的, 我们需要自己在内存里构建出来关系
	cadvisorPodCGroupInfo struct {
		namespace string
		pod       string
		// sum of usage of filesystem of children cgroup
		diskUsage float64
		pod2      *v1.Pod
	}
	containerStatCache struct {
		stat    *cv1.ContainerStats
		metrics []*model.Metric
	}
)

func (c *cadvisorSysCollector) Name() string {
	return "cadvisor"
}

func NewPodSystemResourceCollector(k8smm *k8smeta.Manager, crii cri.Interface, suffix string, interval time.Duration) common.SysCollector {
	return &cadvisorSysCollector{
		k8smm:     k8smm,
		crii:      crii,
		agentMode: core.AgentModeDaemonset,
		suffix:    suffix,
		interval:  interval,
		stopSig:   util.NewStopSignal(),
	}
}

func (c *cadvisorSysCollector) Stop() {
	c.stopSig.StopAndWait()
}

func (c *cadvisorSysCollector) Start() {
	go func() {
		trg := trigger.WithFixedRate(c.interval, 1*time.Second)
		next := trg.Next(nil)
		timer := time.NewTimer(next.Sub(time.Now()))
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				// 时间要算在上个周期上: 57分1秒算出的数据计在 56分 的时间戳上
				c.collectOnce(next.Add(-(c.interval + 1*time.Second)))

				next = trg.Next(nil)
				timer.Reset(next.Sub(time.Now()))
			case <-c.stopSig.C:
				return
			}
		}
	}()
}

func (c *cadvisorSysCollector) filterSelfNodeCadvisorPod(pods []*v1.Pod) *v1.Pod {
	hostIP := c.k8smm.LocalMeta.HostIP()
	for _, pod := range pods {
		if isCadvisor(pod) && pod.Status.HostIP == hostIP {
			return pod
		}
	}
	return nil
}

func (c *cadvisorSysCollector) collectOnce(alignT time.Time) {
	defer func() {
		c.cache1 = c.cache2
		c.cache2 = nil
	}()

	var cPods []*v1.Pod

	selfNs := c.k8smm.LocalMeta.Namespace()
	if selfNs != v1.NamespaceAll {
		// "" 会被认为是 all namespace
		cPods = c.k8smm.PodMeta.GetPodsByNamespace(selfNs)
	}
	if len(cPods) == 0 {
		cPods = c.k8smm.PodMeta.GetPodsByNamespace(cadvisor)
	}

	if c.agentMode == core.AgentModeDaemonset {
		// daemonset 模式下只关心自己所属物理机的指标
		nodeCadvisorPod := c.filterSelfNodeCadvisorPod(cPods)
		if nodeCadvisorPod != nil {
			cPods = []*v1.Pod{nodeCadvisorPod}
		} else {
			cPods = nil
		}
	}

	if len(cPods) == 0 {
		logger.Warnf("[k8s] [cadvisor] no cadvisor pod in ns [%s, %s]", selfNs, cadvisor)
	}

	var wg sync.WaitGroup

	if c.cache1 == nil {
		c.cache1 = &sync.Map{}
	}
	if c.cache2 == nil {
		c.cache2 = &sync.Map{}
	}
	for _, pod := range cPods {
		if !isCadvisor(pod) {
			continue
		}
		bak := pod
		wg.Add(1)
		util.GoWithRecover(func() {
			defer wg.Done()
			c.collectPodResourcesWithCAdvisor(alignT, bak)
		})
	}

	wg.Wait()
}

const (
	cadvisor = "cadvisor"
)

func isCadvisor(pod *v1.Pod) bool {
	if k8slabels.GetApp(pod.Labels) != cadvisor {
		return false
	}
	if pod.Status.Phase != v1.PodRunning {
		return false
	}
	return true
}

func (c *cadvisorSysCollector) collectPodResourcesWithCAdvisor(alignT time.Time, cAdvisorPod *v1.Pod) {
	var metrics []*model.Metric

	cadvisorIp := cAdvisorPod.Status.PodIP
	url := fmt.Sprintf("http://%s:%d/", cadvisorIp, cadvisorPort)
	cc, err := cadvisorclient.NewClient(url)
	if err != nil {
		logger.Errorz("[cadvisor] NewClient error",
			zap.String("cadvisor", cadvisorIp),
			zap.Error(err))
		return
	}
	mi, err := cc.MachineInfo()
	if err != nil {
		logger.Errorz("[cadvisor] get machine info error",
			zap.String("cadvisor", cadvisorIp),
			zap.Error(err))
		return
	}

	ctrs, err := cc.SubcontainersInfo("", &cv1.ContainerInfoRequest{})
	if err != nil {
		logger.Errorz("[cadvisor] get container info error", zap.String("cadvisorIp", cadvisorIp), zap.Error(err))
		return
	}

	// 处理pod指标
	var cadvisorPodInfoMap = make(map[string]*cadvisorPodCGroupInfo)

	{
		for _, ctr := range ctrs {
			if len(ctr.Subcontainers) == 0 {
				namespace := k8slabels.GetNamespace(ctr.Spec.Labels)
				pod := k8slabels.GetPodName(ctr.Spec.Labels)

				if namespace == "" || pod == "" {
					continue
				}

				parent := filepath.Dir(ctr.Name)
				if _, ok := cadvisorPodInfoMap[parent]; !ok {
					cadvisorPodInfoMap[parent] = &cadvisorPodCGroupInfo{
						namespace: namespace,
						pod:       pod,
					}
				}
			}
		}
	}

	alignTs := alignT.UnixMilli()

	for i := range ctrs {
		ctr := &ctrs[i]
		if len(ctr.Stats) == 0 {
			continue
		}

		var lastStat *containerStatCache

		if i, ok := c.cache1.Load(ctr.Name); ok {
			lastStat = i.(*containerStatCache)
		}

		newStat := ctr.Stats[len(ctr.Stats)-1]

		newStatCache := &containerStatCache{
			stat: newStat,
		}
		c.cache2.Store(ctr.Name, newStatCache)

		if lastStat != nil && lastStat.stat.Timestamp == newStat.Timestamp {
			for _, metric := range lastStat.metrics {
				x := *metric
				x.Timestamp = alignTs
				newStatCache.metrics = append(newStatCache.metrics, &x)
			}
			metrics = append(metrics, newStatCache.metrics...)
			// logger.Warnf("[cadvisor] id=[%s] reuse last period size=[%d]", id, len(newStatCache.metrics))
			continue
		}

		if lastStat == nil {
			// logger.Warnf("[cadvisor] id=[%s] last stat is nil", id)
			continue
		}

		s1 := lastStat.stat
		s2 := newStat

		// 两次测量的间隔时间 (单位纳秒)
		deltaTime := s2.Timestamp.Sub(s1.Timestamp) / time.Nanosecond
		if deltaTime == 0 {
			// logger.Warnf("[cadvisor] id=[%s] delta time is 0", id)
			continue
		}

		// 是物理机的 cgroup
		if ctr.Name == "/" {
			newStatCache.metrics = c.collectNode(ctr, cAdvisorPod, newStatCache.metrics, mi, s1, s2, deltaTime, alignTs)
			metrics = append(metrics, newStatCache.metrics...)
			continue
		}

		// 是 pod 级 cgroup
		if cpi, ok := cadvisorPodInfoMap[ctr.Name]; ok {
			newStatCache.metrics = c.collectPodCGroup(ctr, cpi, newStatCache.metrics, mi, s1, s2, deltaTime, alignTs)
			metrics = append(metrics, newStatCache.metrics...)
			continue
		}

		// 是sandbox
		if criCtr, ok := c.crii.GetContainerByCid(ctr.Id); ok && extractor.PodMetaServiceInstance.IsSandbox(criCtr) {
			newStatCache.metrics = c.collectPodSandbox(ctr, newStatCache.metrics, s1, s2, deltaTime, alignTs)
			metrics = append(metrics, newStatCache.metrics...)
			continue
		}

		// 是 container 级 cgroup (叶子 cgroup)
		if len(ctr.Subcontainers) == 0 {
			newStatCache.metrics = c.collectPodContainer(ctr, newStatCache.metrics, mi, s1, s2, deltaTime, alignTs)

			parentCgroup := filepath.Base(ctr.Name)
			if parentCgroup != "" {
				if cpi, ok := cadvisorPodInfoMap[parentCgroup]; ok {
					for _, metric := range newStatCache.metrics {
						if metric.Name == "k8s_container_disk_usage" {
							cpi.diskUsage += metric.Value
							break
						}
					}
				}
			}
		}

		metrics = append(metrics, newStatCache.metrics...)
	}

	for _, cpi := range cadvisorPodInfoMap {
		if cpi.pod2 != nil {
			tags := meta.ExtractPodCommonTags(cpi.pod2)
			metrics = append(metrics, &model.Metric{
				Name:      "k8s_pod_disk_usage",
				Tags:      tags,
				Timestamp: alignTs,
				Value:     cpi.diskUsage,
			})
		}
	}

	{
		// TODO 物理机的指标?
		nodeTags := make(map[string]string)
		nodeTags["ip"] = cAdvisorPod.Status.HostIP

		//metrics = append(metrics, &model.Metric{
		//	Name:      "k8s.node.cpu_total_cores",
		//	Tags:      nodeTags,
		//	Timestamp: alignTs,
		//	Value:     float64(mi.NumCores),
		//})
		//metrics = append(metrics, &model.Metric{
		//	Name:      "k8s.node.mem_total",
		//	Tags:      nodeTags,
		//	Timestamp: alignTs,
		//	Value:     float64(mi.MemoryCapacity),
		//})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_containers",
			Tags:      nodeTags,
			Timestamp: alignTs,
			Value:     float64(len(ctrs)),
		})
	}

	if g, err := gateway.Acquire(); err == nil {
		if c.suffix != "" {
			for _, metric := range metrics {
				if !strings.HasSuffix(metric.Name, c.suffix) {
					metric.Name += c.suffix
				}
			}
		}

		defer gateway.GatewaySingletonHolder.Release()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		resp, err := g.WriteMetricsV1Extension2(ctx, nil, metrics)
		if err != nil || resp.Header.Code != 0 {
			logger.Errorz("[cadvisor] report error", zap.Any("resp", resp), zap.Error(err))
		}
	} else {
		logger.Errorz("[cadvisor] [output] get gateway error", zap.Error(err))
	}
}

func (c *cadvisorSysCollector) collectPodSandbox(ctr *cv1.ContainerInfo, metrics []*model.Metric, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	pod := c.k8smm.PodMeta.GetPodByName(k8slabels.GetNamespace(ctr.Spec.Labels), k8slabels.GetPodName(ctr.Spec.Labels))
	if pod == nil {
		return metrics
	}

	// sandbox 特殊处理
	tags := meta.ExtractPodCommonTags(pod)

	{
		trafficIn, trafficOut, ok := calcTraffic("eth0", s2.Network, s1.Network, deltaTime)
		if ok {

			metrics = append(metrics, &model.Metric{
				Name:      "k8s_pod_traffic_bytin",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     trafficIn,
			})
			metrics = append(metrics, &model.Metric{
				Name:      "k8s_pod_traffic_bytout",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     trafficOut,
			})
		}
	}

	{
		// tcp
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_pod_tcp_established",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Network.Tcp.Established),
		})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_pod_tcp_listen",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Network.Tcp.Listen),
		})
	}
	return metrics
}

func (c *cadvisorSysCollector) collectPodContainer(ctr *cv1.ContainerInfo, metrics []*model.Metric, mi *cv1.MachineInfo, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	pod := c.k8smm.PodMeta.GetPodByName(k8slabels.GetNamespace(ctr.Spec.Labels), k8slabels.GetPodName(ctr.Spec.Labels))
	if pod == nil {
		return metrics
	}

	tags := meta.ExtractPodCommonTags(pod)
	tags["container"] = k8slabels.GetContainerName(ctr.Spec.Labels)

	return c.collectCGroupStats(ctr, metrics, tags, "k8s_container", mi, s1, s2, deltaTime, metricTime, true)
}

func (c *cadvisorSysCollector) collectPodCGroup(ctr *cv1.ContainerInfo, cpi *cadvisorPodCGroupInfo, metrics []*model.Metric, mi *cv1.MachineInfo, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	pod := c.k8smm.PodMeta.GetPodByName(cpi.namespace, cpi.pod)
	if pod == nil {
		return metrics
	}
	cpi.pod2 = pod

	tags := meta.ExtractPodCommonTags(pod)

	return c.collectCGroupStats(ctr, metrics, tags, "k8s_pod", mi, s1, s2, deltaTime, metricTime, false)
}

func (c *cadvisorSysCollector) collectCGroupStats(ctr *cv1.ContainerInfo, metrics []*model.Metric, tags map[string]string, metricPrefix string, mi *cv1.MachineInfo, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64, includeDisk bool) []*model.Metric {
	limitCpu := float64(mi.NumCores)
	if ctr.Spec.Cpu.Quota > 0 && ctr.Spec.Cpu.Period > 0 {
		limitCpu = float64(ctr.Spec.Cpu.Quota) / float64(ctr.Spec.Cpu.Period)
	}
	limitMem := mi.MemoryCapacity
	if ctr.Spec.Memory.Limit > 0 && ctr.Spec.Memory.Limit < mi.MemoryCapacity {
		limitMem = ctr.Spec.Memory.Limit
	}

	{
		// cpu
		// Cpu.Usage.Total "累计"消耗的CPU时间
		// 消耗2个核4秒 那么值就为8

		// 两次测量的CPU时间增量 (单位纳秒)
		deltaTotal := s2.Cpu.Usage.Total - s1.Cpu.Usage.Total
		deltaUser := s2.Cpu.Usage.User - s1.Cpu.Usage.User
		deltaSys := s2.Cpu.Usage.System - s1.Cpu.Usage.System
		if deltaUser+deltaSys > 0 {

			// fmt.Println(s2.Timestamp, float64(deltaTotal)/float64(deltaTime/time.Second)*100)
			// 该公式可以用于算出使用了多少核算力
			// float64(deltaTotal) / float64(deltaTime)

			// 修正不超过100
			utilCpuP := math.Min(float64(deltaTotal)/float64(deltaTime)/limitCpu*100, 100)
			userCpuP := math.Min(float64(deltaTotal)*float64(deltaUser)/float64(deltaUser+deltaSys)/float64(deltaTime)/limitCpu*100, 100)
			sysCpuP := utilCpuP - userCpuP

			// 结果说明 cpu_total 似乎总是比 cpu_user/cpu_sys 的范围小? 咋回事...
			// 所以当3个值放在一起时, cpu_util > cpu_user + cpu_sys

			//fmt.Println(float64(s2.Cpu.Usage.Total-s1.Cpu.Usage.Total) / float64(deltaTime) / limitCpu)
			//fmt.Println(float64(s2.Cpu.Usage.User-s1.Cpu.Usage.User) / float64(deltaTime) / limitCpu)
			//fmt.Println(float64(s2.Cpu.Usage.System-s1.Cpu.Usage.System) / float64(deltaTime) / limitCpu)
			//fmt.Println(float64(int64(s2.Cpu.Usage.Total-s1.Cpu.Usage.Total)-int64(s2.Cpu.Usage.User-s1.Cpu.Usage.User+s2.Cpu.Usage.System-s1.Cpu.Usage.System)) / float64(deltaTime) / limitCpu)

			// 已经使用的cpu算力
			metrics = append(metrics, &model.Metric{
				Name:      metricPrefix + "_cpu_inuse_cores",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     float64(deltaTotal) / float64(deltaTime),
			})
			// cpu算力 上限
			metrics = append(metrics, &model.Metric{
				Name:      metricPrefix + "_cpu_total_cores",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     limitCpu,
			})
			metrics = append(metrics, &model.Metric{
				Name:      metricPrefix + "_cpu_util",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     utilCpuP,
			})
			metrics = append(metrics, &model.Metric{
				Name:      metricPrefix + "_cpu_user",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     userCpuP,
			})
			metrics = append(metrics, &model.Metric{
				Name:      metricPrefix + "_cpu_sys",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     sysCpuP,
			})
		}
	}

	{
		// mem
		useMem := s2.Memory.Usage
		metrics = append(metrics, &model.Metric{
			Name:      metricPrefix + "_mem_cache",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Memory.Cache),
		})
		metrics = append(metrics, &model.Metric{
			Name:      metricPrefix + "_mem_rss",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Memory.RSS),
		})
		metrics = append(metrics, &model.Metric{
			Name:      metricPrefix + "_mem_used",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(useMem - s2.Memory.Cache),
		})
		metrics = append(metrics, &model.Metric{
			Name:      metricPrefix + "_mem_total",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(limitMem),
		})
		metrics = append(metrics, &model.Metric{
			Name:      metricPrefix + "_mem_util",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     math.Min(float64(useMem-s2.Memory.Cache)/float64(limitMem)*100, 100),
		})
	}

	// disk
	if includeDisk {
		usage := uint64(0)
		for i := range s2.Filesystem {
			usage += s2.Filesystem[i].Usage
		}
		metrics = append(metrics, &model.Metric{
			Name:      metricPrefix + "_disk_usage",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(usage),
		})
	}
	return metrics
}

func (c *cadvisorSysCollector) collectNode(ctr *cv1.ContainerInfo, cAdvisorPod *v1.Pod, metrics []*model.Metric, mi *cv1.MachineInfo, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	nodeIp := cAdvisorPod.Status.HostIP
	limitCpu := float64(mi.NumCores)
	if ctr.Spec.Cpu.Quota > 0 && ctr.Spec.Cpu.Period > 0 {
		limitCpu = float64(ctr.Spec.Cpu.Quota) / float64(ctr.Spec.Cpu.Period)
	}
	limitMem := mi.MemoryCapacity
	if ctr.Spec.Memory.Limit > 0 && ctr.Spec.Memory.Limit < mi.MemoryCapacity {
		limitMem = ctr.Spec.Memory.Limit
	}
	tags := map[string]string{
		"ip": nodeIp,
	}
	node := c.k8smm.NodeMeta.GetNodeByIp(nodeIp)
	if node != nil {
		tags["name"] = node.Name
		tags["hostname"] = extractor.PodMetaServiceInstance.NodeHostname(node)
		tags["region"] = k8slabels.GetRegion(node.Labels)
		tags["zone"] = k8slabels.GetZone(node.Labels)
		tags["os"] = node.Labels[k8slabels.LabelK8sOs]
		tags["arch"] = node.Labels[k8slabels.LabelK8sArch]
		tags["instanceType"] = node.Labels[k8slabels.LabelK8sNodeInstanceType]

		// 现镜像数
		//metrics = append(metrics, &model.Metric{
		//	Name:      "k8s_node_images",
		//	Tags:      tags,
		//	Timestamp: metricTime,
		//	Value:     float64(len(node.Status.Images)),
		//})

		// TODO 磁盘使用率?
		// cadvisor 工作不是很好, 因为它很多信息都是基于cgroup的, 有很多虚拟cgroup干扰
		// 参考一下prometheus的 https://github.com/prometheus/node_exporter
	}

	{
		trafficIn, trafficOut, ok := calcTraffic("eth0", s2.Network, s1.Network, deltaTime)
		if ok {

			metrics = append(metrics, &model.Metric{
				Name:      "k8s_node_traffic_bytin",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     trafficIn,
			})
			metrics = append(metrics, &model.Metric{
				Name:      "k8s_node_traffic_bytout",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     trafficOut,
			})
		}
	}

	// TODO 不太准
	{
		// tcp
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_tcp_established",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Network.Tcp.Established),
		})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_tcp_listen",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Network.Tcp.Listen),
		})
	}

	{
		// cpu
		// Cpu.Usage.Total "累计"消耗的CPU时间
		// 消耗2个核4秒 那么值就为8

		// 两次测量的CPU时间增量 (单位纳秒)
		deltaTotal := s2.Cpu.Usage.Total - s1.Cpu.Usage.Total
		deltaUser := s2.Cpu.Usage.User - s1.Cpu.Usage.User
		deltaSys := s2.Cpu.Usage.System - s1.Cpu.Usage.System
		if deltaUser+deltaSys > 0 {

			// fmt.Println(s2.Timestamp, float64(deltaTotal)/float64(deltaTime/time.Second)*100)
			// 该公式可以用于算出使用了多少核算力
			// float64(deltaTotal) / float64(deltaTime)

			// 修正不超过100
			utilCpuP := math.Min(float64(deltaTotal)/float64(deltaTime)/limitCpu*100, 100)
			userCpuP := math.Min(float64(deltaTotal)*float64(deltaUser)/float64(deltaUser+deltaSys)/float64(deltaTime)/limitCpu*100, 100)
			sysCpuP := utilCpuP - userCpuP

			// 结果说明 cpu_total 似乎总是比 cpu_user/cpu_sys 的范围小? 咋回事...
			// 所以当3个值放在一起时, cpu_util > cpu_user + cpu_sys

			//fmt.Println(float64(s2.Cpu.Usage.Total-s1.Cpu.Usage.Total) / float64(deltaTime) / limitCpu)
			//fmt.Println(float64(s2.Cpu.Usage.User-s1.Cpu.Usage.User) / float64(deltaTime) / limitCpu)
			//fmt.Println(float64(s2.Cpu.Usage.System-s1.Cpu.Usage.System) / float64(deltaTime) / limitCpu)
			//fmt.Println(float64(int64(s2.Cpu.Usage.Total-s1.Cpu.Usage.Total)-int64(s2.Cpu.Usage.User-s1.Cpu.Usage.User+s2.Cpu.Usage.System-s1.Cpu.Usage.System)) / float64(deltaTime) / limitCpu)

			// 已经使用的cpu算力
			metrics = append(metrics, &model.Metric{
				Name:      "k8s_node_cpu_inuse_cores",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     float64(deltaTotal) / float64(deltaTime),
			})
			// cpu算力 上限
			metrics = append(metrics, &model.Metric{
				Name:      "k8s_node_cpu_total_cores",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     limitCpu,
			})
			metrics = append(metrics, &model.Metric{
				Name:      "k8s_node_cpu_util",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     utilCpuP,
			})
			metrics = append(metrics, &model.Metric{
				Name:      "k8s_node_cpu_user",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     userCpuP,
			})
			metrics = append(metrics, &model.Metric{
				Name:      "k8s_node_cpu_sys",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     sysCpuP,
			})
		}
	}

	{
		// mem
		useMem := s2.Memory.Usage
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_mem_cache",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Memory.Cache),
		})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_mem_rss",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Memory.RSS),
		})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_mem_used",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(useMem - s2.Memory.Cache),
		})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_mem_total",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(limitMem),
		})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_mem_util",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     math.Min(float64(useMem-s2.Memory.Cache)/float64(limitMem)*100, 100),
		})
	}

	//{
	//	// processes
	//	metrics = append(metrics, &model.Metric{
	//		Name:      "k8s.container.processes_threads",
	//		Tags:      tags,
	//		Timestamp: metricTime,
	//		Value:     float64(s2.Processes.ThreadsCurrent),
	//	})
	//}
	//
	//{
	//	// TODO 不准
	//	// disk
	//	total := uint64(0)
	//	used := uint64(0)
	//	for _, fsStat := range s2.Filesystem {
	//		total += fsStat.Limit
	//		used += fsStat.Usage
	//	}
	//	metrics = append(metrics, &model.Metric{
	//		Name:      "k8s.container.disk_used",
	//		Tags:      tags,
	//		Timestamp: metricTime,
	//		Value:     float64(used),
	//	})
	//	metrics = append(metrics, &model.Metric{
	//		Name:      "k8s.container.disk_total",
	//		Tags:      tags,
	//		Timestamp: metricTime,
	//		Value:     float64(total),
	//	})
	//	if total > 0 {
	//		metrics = append(metrics, &model.Metric{
	//			Name:      "k8s.container.disk_util",
	//			Tags:      tags,
	//			Timestamp: metricTime,
	//			Value:     float64(used) / float64(total) * 100,
	//		})
	//	}
	//}

	return metrics
}

func calcTraffic(ifz string, n2 cv1.NetworkStats, n1 cv1.NetworkStats, deltaTime time.Duration) (float64, float64, bool) {
	for _, s2 := range n2.Interfaces {
		if s2.Name == ifz {

			for _, s1 := range n1.Interfaces {
				if s2.Name == s1.Name {
					bytin := float64(s2.RxBytes-s1.RxBytes) * float64(time.Second) / float64(deltaTime)
					bytout := float64(s2.TxBytes-s1.TxBytes) * float64(time.Second) / float64(deltaTime)
					return bytin, bytout, true
				}
			}

			break
		}
	}
	return 0, 0, false
}
