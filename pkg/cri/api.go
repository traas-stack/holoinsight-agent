/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cri

import (
	"context"
	"io"
)

type (
	Interface interface {
		GetAllPods() []*Pod
		GetContainerByCid(cid string) (*Container, bool)
		CheckSandboxByLabels(labels map[string]string) bool

		// 查询一个pod的信息
		GetPod(ns, pod string) (*Pod, bool)

		GetPodByHostname(hostname string) (*Pod, bool)

		// CopyToContainer copies file to container
		CopyToContainer(ctx context.Context, c *Container, srcPath, dstPath string) error

		// CopyFromContainer copies file from container
		CopyFromContainer(ctx context.Context, c *Container, srcPath, dstPath string) error

		// Exec runs command in target container
		Exec(ctx context.Context, c *Container, req ExecRequest) (ExecResult, error)
	}
	NsEnterType uint8

	// ExecRequest wraps parameters need for exec
	ExecRequest struct {
		Cmd        []string `json:"cmd"`
		Env        []string `json:"env"`
		WorkingDir string   `json:"workingDir"`
		Input      io.Reader
		// User is the user passed to docker exec, defaults to 'root'
		User string
	}
)

const (
	NsEnter_MNT NsEnterType = iota
	NsEnter_NET
	NsEnter_UTS
)
