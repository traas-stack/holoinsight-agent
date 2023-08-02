/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

import (
	"context"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"io/fs"
	"time"
)

// HasCommand checks whether container has specified command
func HasCommand(ctx context.Context, i cri.Interface, c *cri.Container, command string) error {
	//_, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"command", "-v", command}})
	_, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"sh", "-c", fmt.Sprintf("which %s || command -v %s", command, command)}})
	return err
}

func SubContextTimeoutExec(parent context.Context, timeout time.Duration, callback func(context.Context) (cri.ExecResult, error)) (cri.ExecResult, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	return callback(ctx)
}

// ExecMv executes 'mv' command inside container
func ExecMv(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) error {
	_, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"mv", srcPath, dstPath}})
	return err
}

// ExecCp executes 'cp' command inside container
func ExecCp(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) error {
	_, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"cp", srcPath, dstPath}})
	return err
}

// ExecRm executes 'rm' command inside container
func ExecRm(ctx context.Context, i cri.Interface, c *cri.Container, path string) error {
	_, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"rm", path}})
	return err
}

// ExecChmod executes 'chmod' command inside container
func ExecChmod(ctx context.Context, i cri.Interface, c *cri.Container, path string, mode fs.FileMode) error {
	if _, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"chmod", fmt.Sprintf("%#o", mode), path}}); err != nil {
		return err
	}
	return nil
}

func FindFirstPodByIp(i cri.Interface, ip string) *cri.Pod {
	for _, pod := range i.GetAllPods() {
		if pod.IP() == ip {
			return pod
		}
	}
	return nil
}
