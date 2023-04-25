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

		// 复制内容到容器里
		CopyToContainer(ctx context.Context, c *Container, srcPath, dstPath string) error
		// 同步地执行命令, 一般是简单命令, 可以快速返回, 并且返回结果较小
		// 返回值是 exitcode,stdout,stdout,error
		ExecSync(ctx context.Context, c *Container, cmd []string, env []string, workingDir string, input io.Reader) (ExecResult, error)
		// 切ns 然后执行命令
		NsEnterExec(ctx context.Context, nsEnterTypes []NsEnterType, c *Container, cmd []string, env []string, workingDir string, input io.Reader) (ExecResult, error)
		// TODO 调研 nsenter 能否成为标准方式
		NsEnterHelperExec(ctx context.Context, c *Container, args []string, env []string, workingDir string, input interface{}) (ExecResult, error)
	}
	NsEnterType uint8
)

const (
	NsEnter_MNT NsEnterType = iota
	NsEnter_NET
	NsEnter_UTS
)
