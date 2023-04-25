/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package containerlogs

import (
	"context"
	"github.com/docker/docker/api/types"
	dockersdk "github.com/docker/docker/client"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"time"
)

type (
	Manager struct {
		m      *k8s.Manager
		docker *dockersdk.Client
	}
)

func NewManager(m *k8s.Manager) *Manager {
	return &Manager{
		m: m,
	}
}

func (m *Manager) Start() {
	go func() {
		docker, err := dockersdk.NewClientWithOpts(dockersdk.WithHost("unix:///rootfs/var/run/docker.sock"))
		if err != nil {
			panic(err)
		}
		m.docker = docker

		for range time.NewTicker(10 * time.Second).C {
			// TODO docker 有api可以尽量保持兼容吗?
			// 方案1: 定期 docker.containerList + docker.containerInspect
			// 方案2: k8s watch pod 变化 + docker.containerInspect . 有个问题: k8s只能感知pod变化? 能否感知container变化?(比如删除一个container)
			// TODO 能实时同步docker的元信息吗?

			begin := time.Now()
			ctrs, _ := docker.ContainerList(context.Background(), types.ContainerListOptions{})
			end := time.Now()
			logger.Infof("[docker] container size=[%d] cost=[%v]", len(ctrs), end.Sub(begin))

			begin = time.Now()
			for _, ctr := range ctrs {
				docker.ContainerInspect(context.Background(), ctr.ID)
			}
			end = time.Now()
			logger.Infof("[docker] container details cost=[%v]", end.Sub(begin))
		}
	}()
}
