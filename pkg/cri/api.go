/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cri

import (
	"context"
	"io"
	v1 "k8s.io/api/core/v1"
)

type (
	// LocalAgentMeta providers local agent basic info
	LocalAgentMeta interface {
		Namespace() string
		PodName() string
		PodIP() string
		NodeName() string
		NodeIP() string
		Node() *v1.Node
		IsLocalPod(pod *v1.Pod) bool
	}

	MetaStore interface {
		// GetAllPods returns all local pods
		GetAllPods() []*Pod

		// GetPod queries one local pod by namespace and podName.
		// Returns an error if not found.
		GetPod(namespace, podName string) (*Pod, error)

		// GetContainerByCid queries one container by cid
		GetContainerByCid(cid string) (*Container, bool)

		LocalAgentMeta() LocalAgentMeta

		Start() error

		Stop()

		AddListener(MetaListener)
		RemoveListener(MetaListener)
	}
	MetaListener interface {
		OnAnyPodChanged()
	}

	// Interface is the entrance of accessing pod/container meta and executing container commands.
	Interface interface {
		MetaStore

		// CopyToContainer copies file to container
		CopyToContainer(ctx context.Context, c *Container, srcPath, dstPath string) error

		// CopyFromContainer copies file from container
		CopyFromContainer(ctx context.Context, c *Container, srcPath, dstPath string) error

		// Exec runs command in target container
		Exec(ctx context.Context, c *Container, req ExecRequest) (ExecResult, error)

		// ExecAsync runs command in target container
		ExecAsync(ctx context.Context, c *Container, req ExecRequest) (ExecAsyncResult, error)

		Start() error

		Stop()

		// Engine returns the underlying ContainerEngine
		Engine() ContainerEngine
	}

	// ContainerEngine is an abstraction for docker/pouch/containerd
	ContainerEngine interface {
		// Init initializes the engine
		Init() error

		// Type returns engine type, such as "docker" "containerd"
		Type() string

		ListAllContainers(ctx context.Context) ([]*EngineSimpleContainer, error)

		GetContainerDetail(ctx context.Context, cid string) (*EngineDetailContainer, error)

		Exec(ctx context.Context, c *Container, req ExecRequest) (ExecResult, error)

		// ExecAsync runs command in target container
		ExecAsync(ctx context.Context, c *Container, req ExecRequest) (ExecAsyncResult, error)

		// CopyToContainer copies file from src(in agent) to dst(in container)
		CopyToContainer(ctx context.Context, c *Container, src, dst string) error

		// CopyFromContainer copies file from src(in container) to dst(in agent)
		CopyFromContainer(ctx context.Context, c *Container, src, dst string) error

		// Supports checks if engine supports the specified feature
		Supports(feature ContainerEngineFeature) bool
	}

	ContainerEngineFeature uint8

	EngineSimpleContainer struct {
		ID     string
		Labels map[string]string
		Source interface{}
	}

	// EngineDetailContainer Contains the most common container properties, provided by almost all advanced container runtimes.
	EngineDetailContainer struct {
		ID          string
		Name        string
		Labels      map[string]string
		Env         []string
		Source      interface{}
		IsSandbox   bool
		SandboxId   string
		Hostname    string
		Runtime     string
		NetworkMode string
		MergedDir   string
		Mounts      []*MountPoint
		State       ContainerState
	}
	NsEnterType uint8

	// ExecRequest wraps parameters need for exec
	ExecRequest struct {
		Cmd        []string `json:"cmd"`
		Env        []string `json:"env"`
		WorkingDir string   `json:"workingDir"`
		Input      io.Reader
		// User is the user passed to docker exec, defaults to 'root'
		User     string
		NoStdErr bool
	}
)

const (
	Runc = "runc"
	Rund = "rund"

	ContainerEngineFeatureCopy ContainerEngineFeature = iota
)
