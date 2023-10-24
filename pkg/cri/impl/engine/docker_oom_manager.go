/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package engine

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockersdk "github.com/docker/docker/client"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/dockerutils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"time"
)

type (
	OOMManager struct {
		CRI        cri.Interface
		Client     *dockersdk.Client
		oomRecoder *oomRecoder
		stopCh     chan struct{}
	}
)

func NewOOMManager(i cri.Interface, client *dockersdk.Client) *OOMManager {
	return &OOMManager{
		CRI:        i,
		Client:     client,
		oomRecoder: newOOMRecorder(),
		stopCh:     make(chan struct{}),
	}
}

func (m *OOMManager) Start() {
	go m.listenDockerLoop()
	go m.emitLoop()
}

func (m *OOMManager) Stop() {
	close(m.stopCh)
}

func (m *OOMManager) isStopped() bool {
	select {
	case <-m.stopCh:
		return true
	default:
		return false
	}
}

func (m *OOMManager) listenDockerLoop() {
	filter := filters.NewArgs()

	filter.Add("type", "container")

	// We are only interested in the following events
	// container started
	filter.Add("event", "start")
	// container exited
	filter.Add("event", "die")
	// container OOM
	filter.Add("event", "oom")

	for {
		if m.isStopped() {
			return
		}

		func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			logger.Criz("[digest] listen to docker events")
			msgCh, errCh := m.Client.Events(ctx, types.EventsOptions{
				Filters: filter,
			})
			for {
				select {
				case msg := <-msgCh:
					action := dockerutils.ExtractEventAction(msg.Action)
					logger.Criz("[event]", zap.String("cid", msg.ID), zap.String("action", action), zap.Any("msg", msg))
					if action == "oom" {
						m.handleOOM(msg)
					}
				case err := <-errCh:
					logger.Metaz("[event] error", zap.Error(err))
					// slow down
					time.Sleep(time.Second)
					return
				case <-m.stopCh:
					return
				}
			}
		}()
	}
}

func (m *OOMManager) handleOOM(msg events.Message) {
	ctr, ok := m.CRI.GetContainerByCid(msg.ID)
	if !ok || ctr.Sandbox {
		// When oom, container and its sandbox all emit oom
		return
	}

	logger.Metaz("[oom]",
		zap.String("ns", ctr.Pod.Namespace),
		zap.String("pod", ctr.Pod.Name),
		zap.String("container", ctr.K8sContainerName),
		zap.Any("msg", msg))

	m.oomRecoder.add(ctr)
}

func (m *OOMManager) emitLoop() {
	timer, emitTime := util.NewAlignedTimer(time.Minute, 2*time.Second, true, false)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			m.emitOOMMetrics(emitTime)
			emitTime = timer.Next()
		case <-m.stopCh:
			return
		}
	}
}

func (m *OOMManager) emitOOMMetrics(emitTime time.Time) {
	record := m.oomRecoder.getAndClear()

	// k8s_pod_oom
	var metrics []*model.Metric

	processed := make(map[k8stypes.UID]struct{})
	for _, item := range record {
		tags := meta.ExtractContainerCommonTags(item.container)
		processed[item.container.Pod.UID] = struct{}{}

		metrics = append(metrics, &model.Metric{
			Name:      "k8s_pod_oom",
			Tags:      tags,
			Timestamp: emitTime.UnixMilli(),
			Value:     float64(item.count),
		})
	}
	pods := m.CRI.GetAllPods()
	for _, pod := range pods {
		if _, ok := processed[pod.UID]; ok {
			continue
		}
		processed[pod.UID] = struct{}{}
		tags := meta.ExtractPodCommonTags(pod.Pod)
		tags["container"] = "-"
		metrics = append(metrics, &model.Metric{
			Name:      "k8s_pod_oom",
			Tags:      tags,
			Timestamp: emitTime.UnixMilli(),
			Value:     float64(0),
		})
	}

	// TODO Decoupling data production and consumption
	begin := time.Now()
	err := gateway.GetWriteService().WriteV1(context.Background(), &gateway.WriteV1Request{
		Batch: metrics,
	})
	cost := time.Now().Sub(begin)
	if err == nil {
		logger.Infoz("[oom] report success", zap.Int("metrics", len(metrics)), zap.Duration("cost", cost))
	} else {
		logger.Errorz("[oom] report error", zap.Int("metrics", len(metrics)), zap.Duration("cost", cost), zap.Error(err))
	}
}
