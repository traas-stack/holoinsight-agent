/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cadvisor

import (
	"context"
	"fmt"
	cadvisorclient "github.com/google/cadvisor/client"
	cv1 "github.com/google/cadvisor/info/v1"
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
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"math"
	"path/filepath"
	"strings"
	"time"
)

const (
	cadvisorPort = 8080
	cadvisor     = "cadvisor"
)

type (
	cadvisorSysCollector struct {
		k8smm    *k8smeta.K8sLocalMetaManager
		crii     cri.Interface
		suffix   string
		interval time.Duration
		stopSig  *util.StopSignal
		state    *internalState
	}
	// pod cgroup 级别的labels上是没有相关标签的, 我们需要自己在内存里构建出来关系
	cadvisorPodCGroupInfo struct {
		namespace string
		pod       string
		// sum of usage of filesystem of children cgroup
		diskUsage float64
	}
	containerStatCache struct {
		stat    *cv1.ContainerStats
		metrics []*model.Metric
	}
	internalState struct {
		cache1 map[string]*containerStatCache
		cache2 map[string]*containerStatCache
	}
)

func (c *cadvisorSysCollector) Name() string {
	return "cadvisor"
}

func NewPodSystemResourceCollector(k8smm *k8smeta.K8sLocalMetaManager, crii cri.Interface, suffix string, interval time.Duration) common.SysCollector {
	return &cadvisorSysCollector{
		k8smm:    k8smm,
		crii:     crii,
		suffix:   suffix,
		interval: interval,
		stopSig:  util.NewStopSignal(),
		state:    &internalState{},
	}
}

func (c *cadvisorSysCollector) Stop() {
	c.stopSig.StopAndWait()
}

func (c *cadvisorSysCollector) Start() {
	go c.taskLoop()
}

func (c *cadvisorSysCollector) taskLoop() {
	defer c.stopSig.StopDone()

	timer, emitTime := util.NewAlignedTimer(c.interval, time.Second, false, false)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// executes at {interval+1s, 2*interval+1s, ...}
			c.collectOnce(emitTime.Truncate(c.interval).Add(-c.interval))
			emitTime = timer.Next()
		case <-c.stopSig.C:
			return
		}
	}
}

func (c *cadvisorSysCollector) collectOnce(alignT time.Time) {
	begin := time.Now()

	defer func() {
		c.state.cache1 = c.state.cache2
		c.state.cache2 = nil
	}()

	cadvisorPod := c.findCadvisorPod()
	if cadvisorPod == nil {
		logger.Warnf("[k8s] [cadvisor] no cadvisor pod in local node")
		return
	}

	if c.state.cache1 == nil {
		c.state.cache1 = make(map[string]*containerStatCache)
	}
	if c.state.cache2 == nil {
		c.state.cache2 = make(map[string]*containerStatCache)
	}
	err := c.collectOnce0(alignT, cadvisorPod)
	cost := time.Now().Sub(begin)

	logger.Infoz("[cadvisor] collect once done", zap.Duration("cost", cost), zap.Error(err))
}

func (c *cadvisorSysCollector) collectOnce0(metricTime time.Time, cAdvisorPod *v1.Pod) error {
	var metrics []*model.Metric

	url := fmt.Sprintf("http://%s:%d/", cAdvisorPod.Status.PodIP, cadvisorPort)

	cc, err := cadvisorclient.NewClient(url)
	if err != nil {
		return err
	}

	mi, err := cc.MachineInfo()
	if err != nil {
		return err
	}

	ctrs, err := cc.SubcontainersInfo("", &cv1.ContainerInfoRequest{})
	if err != nil {
		return err
	}

	podCGroupInfo := c.makePodCGroupInfo(ctrs)

	alignTs := metricTime.UnixMilli()
	for i := range ctrs {
		ctr := &ctrs[i]
		if len(ctr.Stats) == 0 {
			continue
		}

		var lastStat *containerStatCache

		if x, ok := c.state.cache1[ctr.Name]; ok {
			lastStat = x
		}

		newStat := ctr.Stats[len(ctr.Stats)-1]

		newStatCache := &containerStatCache{
			stat: newStat,
		}
		c.state.cache2[ctr.Name] = newStatCache

		// lastStat == nil means that we haven't run enough for two cycles.
		if lastStat == nil {
			continue
		}

		// 'lastStat.stat.Timestamp == newStat.Timestamp' means that: It means that we call too frequently, exceeding the collection frequency of cadvisor.
		// Or there is a problem inside cadvisor.
		// At this time, we will this period with the data of the previous period.
		if lastStat != nil && lastStat.stat.Timestamp == newStat.Timestamp {
			for _, metric := range lastStat.metrics {
				x := *metric
				x.Timestamp = alignTs
				newStatCache.metrics = append(newStatCache.metrics, &x)
			}
			metrics = append(metrics, newStatCache.metrics...)
			continue
		}

		s1 := lastStat.stat
		s2 := newStat

		deltaTime := s2.Timestamp.Sub(s1.Timestamp) / time.Nanosecond
		if deltaTime == 0 {
			continue
		}

		// '/' is the node cgroup
		if ctr.Name == "/" {
			newStatCache.metrics = c.collectNode(ctr, cAdvisorPod, newStatCache.metrics, mi, s1, s2, deltaTime, alignTs)
			metrics = append(metrics, newStatCache.metrics...)
			continue
		}

		// pod level cgroup
		if cpi, ok := podCGroupInfo[ctr.Name]; ok {
			newStatCache.metrics = c.collectPodCGroup(ctr, cpi, newStatCache.metrics, mi, s1, s2, deltaTime, alignTs)
			metrics = append(metrics, newStatCache.metrics...)
			continue
		}

		// sandbox: it holds network traffic metrics
		if criCtr, ok := c.crii.GetContainerByCid(ctr.Id); ok && criCtr.Sandbox {
			newStatCache.metrics = c.collectPodSandbox(ctr, newStatCache.metrics, s1, s2, deltaTime, alignTs)
			metrics = append(metrics, newStatCache.metrics...)
			continue
		}

		// container cgroup
		if len(ctr.Subcontainers) == 0 {
			newStatCache.metrics = c.collectPodContainer(ctr, newStatCache.metrics, mi, s1, s2, deltaTime, alignTs)
		}

		metrics = append(metrics, newStatCache.metrics...)
	}

	// TODO Inaccurate
	//for _, cpi := range podCGroupInfo {
	//	if cpi.pod2 != nil {
	//		tags := meta.ExtractPodCommonTags(cpi.pod2)
	//		metrics = append(metrics, &model.Metric{
	//			Name:      "k8s_pod_disk_usage",
	//			Tags:      tags,
	//			Timestamp: alignTs,
	//			Value:     cpi.diskUsage,
	//		})
	//	}
	//}

	{
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

	return nil
}

func (c *cadvisorSysCollector) collectPodSandbox(ctr *cv1.ContainerInfo, metrics []*model.Metric, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	pod := c.k8smm.LocalPodMeta.GetPodByName(k8slabels.GetNamespace(ctr.Spec.Labels), k8slabels.GetPodName(ctr.Spec.Labels))
	if pod == nil {
		return metrics
	}

	tags := meta.ExtractPodCommonTags(pod)

	// traffic
	{
		// TODO hardcode 'eth0'. Should we add up the traffic of all interfaces?
		if trafficIn, trafficOut, ok := calcTraffic("eth0", s2.Network, s1.Network, deltaTime); ok {
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

	// tcp
	{
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
	pod := c.k8smm.LocalPodMeta.GetPodByName(k8slabels.GetNamespace(ctr.Spec.Labels), k8slabels.GetPodName(ctr.Spec.Labels))
	if pod == nil {
		return metrics
	}

	tags := meta.ExtractPodCommonTags(pod)
	tags["container"] = k8slabels.GetContainerName(ctr.Spec.Labels)

	// TODO This data is not very useful now, and the server does not display it
	return c.collectCGroupStats(ctr, metrics, tags, "k8s_container", mi, s1, s2, deltaTime, metricTime, true)
}

func (c *cadvisorSysCollector) collectPodCGroup(ctr *cv1.ContainerInfo, cpi *cadvisorPodCGroupInfo, metrics []*model.Metric, mi *cv1.MachineInfo, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	pod := c.k8smm.LocalPodMeta.GetPodByName(cpi.namespace, cpi.pod)
	if pod == nil {
		return metrics
	}

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

	// cpu
	{
		deltaTotal := s2.Cpu.Usage.Total - s1.Cpu.Usage.Total
		deltaUser := s2.Cpu.Usage.User - s1.Cpu.Usage.User
		deltaSys := s2.Cpu.Usage.System - s1.Cpu.Usage.System
		// deltaOther := deltaTotal - deltaUser - deltaSys
		if deltaUser+deltaSys > 0 {

			utilCpuP := math.Min(float64(deltaTotal)/float64(deltaTime)/limitCpu*100, 100)
			userCpuP := math.Min(float64(deltaUser)/float64(deltaTime)/limitCpu*100, 100)
			sysCpuP := math.Min(float64(deltaSys)/float64(deltaTime)/limitCpu*100, 100)

			metrics = append(metrics, &model.Metric{
				Name:      metricPrefix + "_cpu_inuse_cores",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     float64(deltaTotal) / float64(deltaTime),
			})
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

	// mem
	{
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

	// TODO Inaccurate
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
	limitCpu := float64(mi.NumCores)
	if ctr.Spec.Cpu.Quota > 0 && ctr.Spec.Cpu.Period > 0 {
		limitCpu = float64(ctr.Spec.Cpu.Quota) / float64(ctr.Spec.Cpu.Period)
	}

	limitMem := mi.MemoryCapacity
	if ctr.Spec.Memory.Limit > 0 && ctr.Spec.Memory.Limit < mi.MemoryCapacity {
		limitMem = ctr.Spec.Memory.Limit
	}

	tags := c.extractNodeTags(cAdvisorPod)

	// network traffic
	{
		if trafficIn, trafficOut, ok := calcTraffic("eth0", s2.Network, s1.Network, deltaTime); ok {
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

	// tcp
	{
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

	// TODO code duplicated
	// cpu
	{
		deltaTotal := s2.Cpu.Usage.Total - s1.Cpu.Usage.Total
		deltaUser := s2.Cpu.Usage.User - s1.Cpu.Usage.User
		deltaSys := s2.Cpu.Usage.System - s1.Cpu.Usage.System
		// TODO What exactly does deltaOther include?
		// deltaOther := deltaTotal - deltaUser - deltaSys
		if deltaUser+deltaSys > 0 {

			// TODO There seems to be some problems with this calculation, and the relationship between util, user, and sys has not been clarified.
			utilCpuP := math.Min(float64(deltaTotal)/float64(deltaTime)/limitCpu*100, 100)
			userCpuP := math.Min(float64(deltaUser)/float64(deltaTime)/limitCpu*100, 100)
			sysCpuP := math.Min(float64(deltaSys)/float64(deltaTime)/limitCpu*100, 100)
			//userCpuP := math.Min(float64(deltaTotal)*float64(deltaUser)/float64(deltaUser+deltaSys)/float64(deltaTime)/limitCpu*100, 100)
			//sysCpuP := utilCpuP - userCpuP

			metrics = append(metrics, &model.Metric{
				Name:      "k8s_node_cpu_inuse_cores",
				Tags:      tags,
				Timestamp: metricTime,
				Value:     float64(deltaTotal) / float64(deltaTime),
			})

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

	// mem
	{
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

	return metrics
}

// findCadvisorPod returns cadvisor pod in current node
func (c *cadvisorSysCollector) findCadvisorPod() *v1.Pod {
	pods := c.k8smm.LocalPodMeta.GetPodsByNamespace(c.k8smm.LocalAgentMeta.Namespace())
	for _, pod := range pods {
		if k8slabels.GetApp(pod.Labels) != cadvisor {
			continue
		}
		if pod.Status.Phase != v1.PodRunning {
			continue
		}
		return pod
	}
	return nil
}

// makePodCGroupInfo builds a map, whose key is pod level cgroup path
func (c *cadvisorSysCollector) makePodCGroupInfo(ctrs []cv1.ContainerInfo) map[string]*cadvisorPodCGroupInfo {
	cadvisorPodInfoMap := make(map[string]*cadvisorPodCGroupInfo)
	for _, ctr := range ctrs {
		if len(ctr.Subcontainers) == 0 {
			namespace := k8slabels.GetNamespace(ctr.Spec.Labels)
			pod := k8slabels.GetPodName(ctr.Spec.Labels)

			// skip non k8s container
			if namespace == "" || pod == "" {
				continue
			}

			// container group: /kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod5c51e64f_4bde_4f55_bdc0_6067bad00435.slice/docker-2937209ace573c40e882b8781f9256d7e2c0a94071784613bc634a8ba72f885c.scope
			// pod cgroup: /kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod5c51e64f_4bde_4f55_bdc0_6067bad00435.slice
			parent := filepath.Dir(ctr.Name)
			if _, ok := cadvisorPodInfoMap[parent]; !ok {
				cadvisorPodInfoMap[parent] = &cadvisorPodCGroupInfo{
					namespace: namespace,
					pod:       pod,
				}
			}
		}
	}

	return cadvisorPodInfoMap
}

func (c *cadvisorSysCollector) extractNodeTags(cAdvisorPod *v1.Pod) map[string]string {
	// Prepare node level tags
	tags := map[string]string{
		"ip": cAdvisorPod.Status.HostIP,
	}
	if node := c.k8smm.NodeMeta.GetNodeByIp(cAdvisorPod.Status.HostIP); node != nil {
		tags["name"] = node.Name
		tags["hostname"] = extractor.PodMetaServiceInstance.NodeHostname(node)
		tags["region"] = k8slabels.GetRegion(node.Labels)
		tags["zone"] = k8slabels.GetZone(node.Labels)
		tags["os"] = node.Labels[k8slabels.LabelK8sOs]
		tags["arch"] = node.Labels[k8slabels.LabelK8sArch]
		tags["instanceType"] = node.Labels[k8slabels.LabelK8sNodeInstanceType]
	}
	return tags
}

func calcTraffic(interfaceName string, n2 cv1.NetworkStats, n1 cv1.NetworkStats, deltaTime time.Duration) (float64, float64, bool) {
	for _, s2 := range n2.Interfaces {
		if s2.Name == interfaceName {

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
