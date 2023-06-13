/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cadvisor

import (
	"context"
	"encoding/gob"
	"fmt"
	cadvisorclient "github.com/google/cadvisor/client"
	cv1 "github.com/google/cadvisor/info/v1"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8slabels"
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
	"sync"
	"time"
)

const (
	cadvisorPort = 8080
	cadvisor     = "cadvisor"
)

type (
	cadvisorSysCollector struct {
		cri      cri.Interface
		suffix   string
		interval time.Duration
		stopSig  *util.StopSignal
		mutex    sync.Mutex
		state    internalState
		timer    *util.AlignedTimer
	}
	podCGroupInfo struct {
		namespace string
		pod       string
		// sum of usage of filesystem of children cgroup
		diskUsage float64
	}
	internalState struct {
		Cache1       map[string]*containerStatCache `json:"cache1,omitempty"`
		cache2       map[string]*containerStatCache
		queryCost    time.Duration
		sendCost     time.Duration
		metricsCount int
	}
	containerStatCache struct {
		Stat    *cv1.ContainerStats `json:"stat,omitempty"`
		Metrics []*model.Metric     `json:"metrics,omitempty"`
	}
	cadvisorSysCollectorStateObj struct {
		Cache1     map[string]*containerStatCache `json:"cache1,omitempty"`
		TimerBytes []byte                         `json:"timerBytes"`
	}
)

func init() {
	gob.Register(&cadvisorSysCollectorStateObj{})
}

func (c *cadvisorSysCollector) SaveState() (interface{}, error) {
	timerBytes, err := c.timer.SaveState()
	if err != nil {
		return nil, err
	}
	return &cadvisorSysCollectorStateObj{
		Cache1:     c.state.Cache1,
		TimerBytes: timerBytes,
	}, nil
}

func (c *cadvisorSysCollector) LoadState(i interface{}) error {
	if i == nil {
		return nil
	}

	state := i.(*cadvisorSysCollectorStateObj)

	c.state.Cache1 = state.Cache1

	if err := c.timer.LoadState(state.TimerBytes); err != nil {
		return err
	}

	return nil
}

func (c *cadvisorSysCollector) Name() string {
	return "cadvisor"
}

func NewPodSystemResourceCollector(cri cri.Interface, suffix string, interval time.Duration) common.SysCollector {
	timer, _ := util.NewAlignedTimer(interval, time.Second, false, false)

	return &cadvisorSysCollector{
		cri:      cri,
		suffix:   suffix,
		interval: interval,
		stopSig:  util.NewStopSignal(),
		timer:    timer,
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

	timer := c.timer
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// executes at {interval+1s, 2*interval+1s, ...}
			func() {
				c.mutex.Lock()
				defer c.mutex.Unlock()
				if c.stopSig.IsStopAsked() {
					return
				}
				metricTime := timer.NextEmitTime().Truncate(c.interval).Add(-c.interval)
				c.collectOnce(metricTime)
				timer.Next()
			}()
		case <-c.stopSig.C:
			return
		}
	}
}

func (c *cadvisorSysCollector) collectOnce(alignT time.Time) {
	begin := time.Now()

	defer func() {
		c.state.Cache1 = c.state.cache2
		c.state.cache2 = nil
	}()

	cadvisorPod := c.findCadvisorPod()
	if cadvisorPod == nil {
		logger.Warnf("[k8s] [cadvisor] no cadvisor pod in local node")
		return
	}

	if c.state.Cache1 == nil {
		c.state.Cache1 = make(map[string]*containerStatCache)
	}
	if c.state.cache2 == nil {
		c.state.cache2 = make(map[string]*containerStatCache)
	}
	c.state.metricsCount = 0
	c.state.queryCost = 0
	c.state.sendCost = 0
	err := c.collectOnce0(alignT, cadvisorPod)
	cost := time.Now().Sub(begin)

	logger.Infoz("[cadvisor] collect once done",
		zap.Int("metrics", c.state.metricsCount),     //
		zap.Duration("queryCost", c.state.queryCost), //
		zap.Duration("sendCost", c.state.sendCost),   //
		zap.Duration("cost", cost),
		zap.Error(err))
}

func (c *cadvisorSysCollector) collectOnce0(metricTime time.Time, cAdvisorPod *v1.Pod) error {

	url := fmt.Sprintf("http://%s:%d/", cAdvisorPod.Status.PodIP, cadvisorPort)

	cc, err := cadvisorclient.NewClient(url)
	if err != nil {
		return err
	}

	queryBegin := time.Now()
	mi, err := cc.MachineInfo()
	if err != nil {
		return err
	}
	// query latest stat of every container
	ctrs, err := cc.SubcontainersInfo("", &cv1.ContainerInfoRequest{
		NumStats: 1,
	})
	if err != nil {
		return err
	}
	c.state.queryCost = time.Now().Sub(queryBegin)

	metrics := c.calcMetrics(metricTime, cAdvisorPod, mi, ctrs)

	sendBegin := time.Now()
	err = c.send(metrics)
	c.state.sendCost = time.Now().Sub(sendBegin)

	return err
}

func (c *cadvisorSysCollector) collectPodSandbox(ctr *cv1.ContainerInfo, metrics []*model.Metric, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	pod, err := c.cri.GetPod(k8slabels.GetNamespace(ctr.Spec.Labels), k8slabels.GetPodName(ctr.Spec.Labels))
	if err != nil {
		return metrics
	}

	tags := meta.ExtractPodCommonTags(pod.Pod)

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
			Value:     float64(s2.Network.Tcp.Established + s2.Network.Tcp6.Established),
		})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_pod_tcp_listen",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Network.Tcp.Listen + s2.Network.Tcp6.Listen),
		})
	}

	return metrics
}

func (c *cadvisorSysCollector) collectPodContainer(ctr *cv1.ContainerInfo, metrics []*model.Metric, mi *cv1.MachineInfo, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	pod, err := c.cri.GetPod(k8slabels.GetNamespace(ctr.Spec.Labels), k8slabels.GetPodName(ctr.Spec.Labels))
	if err != nil {
		return metrics
	}

	tags := meta.ExtractPodCommonTags(pod.Pod)
	tags["container"] = k8slabels.GetContainerName(ctr.Spec.Labels)

	// TODO This data is not very useful now, and the server does not display it
	return c.collectCGroupStats(ctr, metrics, tags, "k8s_container", mi, s1, s2, deltaTime, metricTime, true)
}

func (c *cadvisorSysCollector) collectPodCGroup(ctr *cv1.ContainerInfo, cpi *podCGroupInfo, metrics []*model.Metric, mi *cv1.MachineInfo, s1 *cv1.ContainerStats, s2 *cv1.ContainerStats, deltaTime time.Duration, metricTime int64) []*model.Metric {
	pod, e := c.cri.GetPod(cpi.namespace, cpi.pod)
	if e != nil {
		return metrics
	}

	tags := meta.ExtractPodCommonTags(pod.Pod)

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
			Value:     float64(s2.Network.Tcp.Established + s2.Network.Tcp6.Established),
		})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_tcp_listen",
			Tags:      tags,
			Timestamp: metricTime,
			Value:     float64(s2.Network.Tcp.Listen + s2.Network.Tcp6.Listen),
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
	agentNamespace := c.cri.LocalAgentMeta().Namespace()
	pods := c.cri.GetAllPods()
	for _, pod := range pods {
		if pod.Namespace != agentNamespace {
			continue
		}
		if k8slabels.GetApp(pod.Labels) != cadvisor {
			continue
		}
		if pod.Status.Phase != v1.PodRunning {
			continue
		}
		return pod.Pod
	}
	return nil
}

// makePodCGroupInfo builds a map, whose key is pod level cgroup path
func (c *cadvisorSysCollector) makePodCGroupInfo(ctrs []cv1.ContainerInfo) map[string]*podCGroupInfo {
	cadvisorPodInfoMap := make(map[string]*podCGroupInfo)
	for _, ctr := range ctrs {
		if len(ctr.Subcontainers) == 0 {
			namespace := k8slabels.GetNamespace(ctr.Spec.Labels)
			pod := k8slabels.GetPodName(ctr.Spec.Labels)

			// skip non k8s container
			if namespace == "" || pod == "" {
				continue
			}

			// docker:
			// container cgroup: /kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod5c51e64f_4bde_4f55_bdc0_6067bad00435.slice/docker-2937209ace573c40e882b8781f9256d7e2c0a94071784613bc634a8ba72f885c.scope
			// pod cgroup: /kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod5c51e64f_4bde_4f55_bdc0_6067bad00435.slice

			// containerd:
			// container cgroup /kubepods/podedf083fd-f89d-4f11-8b86-8b24828a54a4/36b3874d38ac7d0636b9badccf265d0eeb5b9db8cae87350aca41a150e98a985
			// pod cgroup /kubepods/podedf083fd-f89d-4f11-8b86-8b24828a54a4

			parent := filepath.Dir(ctr.Name)
			if _, ok := cadvisorPodInfoMap[parent]; !ok {
				cadvisorPodInfoMap[parent] = &podCGroupInfo{
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
	if node := c.cri.LocalAgentMeta().Node(); node != nil {
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

// TODO refactor
func (c *cadvisorSysCollector) send(metrics []*model.Metric) error {
	if c.suffix != "" {
		for _, metric := range metrics {
			if !strings.HasSuffix(metric.Name, c.suffix) {
				metric.Name += c.suffix
			}
		}
	}
	c.state.metricsCount = len(metrics)

	// TODO Decoupling data production and consumption
	return gateway.GetWriteService().WriteV1(context.Background(), &gateway.WriteV1Request{
		Batch: metrics,
	})
}

func (c *cadvisorSysCollector) calcMetrics(metricTime time.Time, cAdvisorPod *v1.Pod, mi *cv1.MachineInfo, ctrs []cv1.ContainerInfo) []*model.Metric {
	alignTs := metricTime.UnixMilli()
	podCGroupInfo := c.makePodCGroupInfo(ctrs)

	var metrics []*model.Metric
	for i := range ctrs {
		ctr := &ctrs[i]
		if len(ctr.Stats) == 0 {
			continue
		}
		var lastStat *containerStatCache

		if x, ok := c.state.Cache1[ctr.Name]; ok {
			lastStat = x
		}

		newStat := ctr.Stats[len(ctr.Stats)-1]

		newStatCache := &containerStatCache{
			Stat: newStat,
		}
		c.state.cache2[ctr.Name] = newStatCache

		// lastStat == nil means that we haven't run enough for two cycles.
		if lastStat == nil {
			continue
		}

		// 'lastStat.stat.Timestamp == newStat.Timestamp' means that: It means that we call too frequently, exceeding the collection frequency of cadvisor.
		// Or there is a problem inside cadvisor.
		// At this time, we will this period with the data of the previous period.
		if lastStat != nil && lastStat.Stat.Timestamp == newStat.Timestamp {
			for _, metric := range lastStat.Metrics {
				x := *metric
				x.Timestamp = alignTs
				newStatCache.Metrics = append(newStatCache.Metrics, &x)
			}
			metrics = append(metrics, newStatCache.Metrics...)
			continue
		}

		s1 := lastStat.Stat
		s2 := newStat

		deltaTime := s2.Timestamp.Sub(s1.Timestamp) / time.Nanosecond
		if deltaTime == 0 {
			continue
		}

		func() {
			defer func() {
				metrics = append(metrics, newStatCache.Metrics...)
			}()

			if ctr.Name == "/" {
				// '/' is the node cgroup
				newStatCache.Metrics = c.collectNode(ctr, cAdvisorPod, newStatCache.Metrics, mi, s1, s2, deltaTime, alignTs)
			} else if cpi, ok := podCGroupInfo[ctr.Name]; ok {
				// pod level cgroup
				newStatCache.Metrics = c.collectPodCGroup(ctr, cpi, newStatCache.Metrics, mi, s1, s2, deltaTime, alignTs)
			} else if criCtr, ok := c.cri.GetContainerByCid(ctr.Id); ok && criCtr.Sandbox {
				// sandbox: it holds network traffic Metrics
				newStatCache.Metrics = c.collectPodSandbox(ctr, newStatCache.Metrics, s1, s2, deltaTime, alignTs)
			} else if len(ctr.Subcontainers) == 0 {
				// container cgroup
				newStatCache.Metrics = c.collectPodContainer(ctr, newStatCache.Metrics, mi, s1, s2, deltaTime, alignTs)
			} else {
				// unknown case
			}
		}()
	}

	{
		nodeTags := make(map[string]string)
		nodeTags["ip"] = cAdvisorPod.Status.HostIP
		//})
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_node_containers",
			Tags:      nodeTags,
			Timestamp: alignTs,
			Value:     float64(len(ctrs)),
		})
	}
	return metrics
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
