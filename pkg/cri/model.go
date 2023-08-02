/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cri

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"io"
	v1 "k8s.io/api/core/v1"
	"sync"
	"time"
)

const (
	DefaultSampleOutputLength = 1024
)

const (
	HackInit = iota
	HackIng
	HackOk
	HackSkipped
	HackRetryError
)

// TODO 我们推出一个规范 让用户按我们规范做 就认为它是主容器
var (
	ErrMultiBiz        = errors.New("multi biz containers")
	ErrNoBiz           = errors.New("no biz container")
	ErrNoSuchContainer = errors.New("no such container")
)

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
	}
	Container struct {
		// container 可能会依赖底层实现, 因此这里不让它依赖具体的实现类
		Id string

		// 所属的pod
		Pod *Pod `json:"-"`

		// ContainerName example:
		// /k8s_POD_xxx
		// /k8s_xxx
		ContainerName string

		// K8sContainerName k8s container name
		K8sContainerName string

		// 常见状态
		State ContainerState

		// 容器常见配置
		Labels map[string]string
		Env    map[string]string

		// 一些我们运行时会用到的值

		// 挂载点 已经转换到hostfs
		Mounts []*MountPoint

		// docker 的 MergedDir 特有字段, 已经转换到hostfs
		MergedDir string

		// 是否是一个 sandbox
		Sandbox bool

		// 该 container 所属的 pod 的 sandbox cid
		SandboxID string

		Tz TzInfo

		// 该容器是否已经被我们hack过
		Hacked int

		// 是否为一个主容器
		MainBiz bool
		Sidecar bool

		// pouch 场景, 只有登录到 container 里 才能获取 daemonset 的 hostname
		Hostname string

		// 运行时
		Runtime string

		// NetworkMode
		NetworkMode string

		// Attributes can be used to prevent arbitrary extension fields
		Attributes sync.Map
	}
	ContainerState struct {
		Pid    int
		Status string
	}

	MountPoint struct {
		// Source is the path in agent (already starts with core.GetHostFs())
		Source string
		// Destination is the container path
		Destination string
		RW          bool
	}
	ExecResult struct {
		Cmd      string
		ExitCode int
		Stdout   *bytes.Buffer
		Stderr   *bytes.Buffer
	}
	ExecAsyncResult struct {
		Cmd    string
		Result chan ExecAsyncResultCode
		Stdout io.Reader
		Stderr io.Reader
	}
	ExecAsyncResultCode struct {
		Code int
		Err  error
	}
	TzInfo struct {
		Name   string
		TzObj  *time.Location `json:"-"`
		Zone   string
		Offset int

		// EtcLocaltime is timezone name read from /etc/localtime
		EtcLocaltime string
		// EnvTz is timezone name read from Env 'TZ'
		EnvTz string
	}
)

// 如果有且只有一个 main biz 就直接返回 否则返回 nil 让用户自己检查去
func (p *Pod) MainBiz() *Container {
	if len(p.Biz) == 1 {
		return p.Biz[0]
	}
	return nil
}

func (p *Pod) MainBizE() (*Container, error) {
	switch len(p.Biz) {
	case 0:
		return nil, ErrNoBiz
	case 1:
		return p.Biz[0], nil
	default:
		return nil, ErrMultiBiz
	}
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

func (p *Pod) GetContainer(k8sContainerName string) (*Container, error) {
	for _, container := range p.All {
		if container.K8sContainerName == k8sContainerName {
			return container, nil
		}
	}
	return nil, ErrNoSuchContainer
}

func (c *Container) IsRunning() bool {
	return c.State.Pid > 0 && c.State.Status == "running"
}

func (c *Container) ShortContainerID() string {
	return ShortContainerId(c.Id)
}

// GetTz returns the time zone used by this container
func (c *Container) GetTz() *time.Location {
	return c.Tz.TzObj
}

func (c *Container) GetTzName() string {
	return c.Tz.Name
}

func NoPodError(ns, pod string) error {
	return fmt.Errorf("no pod ns=[%s] pod=[%s]", ns, pod)
}

func (s *ContainerState) IsRunning() bool {
	return s.Status == "running"
}

func (r *ExecResult) SampleOutput() (stdout string, stderr string) {
	return r.SampleOutputLength(DefaultSampleOutputLength)
}

func (r *ExecResult) SampleOutputLength(length int) (stdout string, stderr string) {
	if r.Stdout != nil {
		stdout = util.SubstringMax(r.Stdout.String(), DefaultSampleOutputLength)
	}
	if r.Stderr != nil {
		stderr = util.SubstringMax(r.Stderr.String(), DefaultSampleOutputLength)
	}
	return
}
