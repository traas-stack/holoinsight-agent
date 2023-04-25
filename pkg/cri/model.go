/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cri

import (
	"bytes"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
)

// TODO 我们推出一个规范 让用户按我们规范做 就认为它是主容器
var ErrMultiBiz = errors.New("multi biz containers")

type (
	Pod struct {
		// 我们依赖k8s的模型是肯定的
		*v1.Pod
		// All containers
		All []*Container `json:"-"`
		// TODO 考虑大小为1, 改名为main
		Biz     []*Container `json:"biz"`
		Sidecar []*Container `json:"sidecar"`
		// Sandbox container
		Sandbox *Container `json:"sandbox"`
		App     string     `json:"app"`
	}
	Container struct {
		// container 可能会依赖底层实现, 因此这里不让它依赖具体的实现类
		Id string

		// 所属的pod
		Pod *Pod `json:"-"`

		Name string

		// 常见状态
		State ContainerState

		// 容器常见配置
		Labels map[string]string
		Env    map[string]string

		// 一些我们运行时会用到的值

		// 标准输出路径, 已经转换到hostfs
		LogPath string

		// 挂载点 已经转换到hostfs
		Mounts []*MountPoint

		// docker 的 MergedDir 特有字段, 已经转换到hostfs
		MergedDir string

		// 是否是一个 sandbox
		Sandbox bool

		// 该 container 所属的 pod 的 sandbox cid
		SandboxID string

		// 系统级别时区 /etc/localtime
		EtcLocaltime string

		// 有一些用户在ENV上设置了TZ变量
		EnvTz string

		// 该容器是否已经被我们hack过
		Hacked bool

		// 是否为一个主容器
		MainBiz bool
		Sidecar bool

		// pouch 场景, 只有登录到 container 里 才能获取 daemonset 的 hostname
		Hostname string

		// 运行时
		Runtime string
	}
	ContainerState struct {
		Pid       int
		StartedAt string
		Status    string
	}

	MountPoint struct {
		// TODO type ?
		// path in host, 已经转换到hostfs
		Source string
		// path in container
		Destination string
	}
	ExecResult struct {
		Cmd      string
		ExitCode int
		Stdout   *bytes.Buffer
		Stderr   *bytes.Buffer
	}
)

// 如果有且只有一个 main biz 就直接返回 否则返回 nil 让用户自己检查去
func (p *Pod) MainBiz() *Container {
	if len(p.Biz) == 1 {
		return p.Biz[0]
	}
	return nil
}

func (p *Pod) IP() string {
	return p.Status.PodIP
}

func (p *Pod) HostIP() string {
	return p.Status.HostIP
}

func (p *Pod) IsRunning() bool {
	return p.Status.Phase == v1.PodRunning
}

func (c *Container) IsRunning() bool {
	return c.State.Status == "running"
}

func NoPodError(ns, pod string) error {
	return fmt.Errorf("no pod ns=[%s] pod=[%s]", ns, pod)
}
