/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package engine

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	dockersdk "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/dockerutils"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/k8slabels"
	k8smetaextractor "github.com/traas-stack/holoinsight-agent/pkg/k8s/k8smeta/extractor"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type (
	// DockerContainerEngine Docker container engine
	DockerContainerEngine struct {
		Client  *dockersdk.Client
		isPouch bool
	}
)

var (
	// Make sure *DockerContainerEngine impl cri.ContainerEngine
	_ cri.ContainerEngine = &DockerContainerEngine{}
)

func (e *DockerContainerEngine) Init() error {
	e.isPouch = strings.HasSuffix(e.Client.DaemonHost(), "pouchd.sock")
	return nil
}

func (e *DockerContainerEngine) Type() string {
	return "docker"
}

func (e *DockerContainerEngine) ListAllContainers(ctx context.Context) ([]*cri.EngineSimpleContainer, error) {
	containers, err := e.Client.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}
	items := make([]*cri.EngineSimpleContainer, len(containers))
	for i := range containers {
		copy := containers[i]
		items[i] = &cri.EngineSimpleContainer{
			ID:     copy.ID,
			Labels: copy.Labels,
			Source: &copy,
		}
	}
	return items, nil
}

func (e *DockerContainerEngine) GetContainerDetail(ctx context.Context, cid string) (*cri.EngineDetailContainer, error) {
	i, err := e.Client.ContainerInspect(ctx, cid)
	if err != nil {
		return nil, err
	}
	detail := &cri.EngineDetailContainer{
		ID:          i.ID,
		Name:        i.Name,
		Labels:      i.Config.Labels,
		Env:         i.Config.Env,
		Source:      i,
		IsSandbox:   k8smetaextractor.PodMetaServiceInstance.Sandbox(k8slabels.GetContainerName(i.Config.Labels), i.Config.Labels),
		SandboxId:   i.Config.Labels["io.kubernetes.sandbox.id"],
		Hostname:    i.Config.Hostname,
		Runtime:     i.HostConfig.Runtime,
		NetworkMode: string(i.HostConfig.NetworkMode),
		MergedDir:   "",
		Mounts:      nil,
		State: cri.ContainerState{
			Pid:    i.State.Pid,
			Status: i.State.Status,
		},
	}

	if detail.Runtime == cri.Runc {
		// overlay2 for docker
		// overlayfs for pouch
		if i.GraphDriver.Name == "overlay2" || i.GraphDriver.Name == "overlayfs" {
			for k, v := range i.GraphDriver.Data {
				if v == "" {
					continue
				}
				switch k {
				case dockerutils.MergedDir:
					// MeredDir now only works in runc runtime.
					detail.MergedDir = v
				}
			}
		}
	}

	for _, m := range i.Mounts {
		detail.Mounts = append(detail.Mounts, &cri.MountPoint{
			Source:      m.Source,
			Destination: m.Destination,
			RW:          m.RW,
		})
	}

	return detail, nil
}

func (e *DockerContainerEngine) Exec(ctx context.Context, c *cri.Container, req cri.ExecRequest) (cri.ExecResult, error) {
	invalidResult := cri.ExecResult{Cmd: strings.Join(req.Cmd, " "), ExitCode: -1}
	create, err := e.Client.ContainerExecCreate(ctx, c.Id, types.ExecConfig{
		User:         req.User,
		Privileged:   false,
		Tty:          false,
		AttachStdin:  req.Input != nil,
		AttachStderr: true,
		AttachStdout: true,
		Detach:       false,
		DetachKeys:   "",
		Env:          req.Env,
		WorkingDir:   req.WorkingDir,
		Cmd:          req.Cmd,
	})
	if err != nil {
		return invalidResult, err
	}

	resp, err := e.Client.ContainerExecAttach(ctx, create.ID, types.ExecStartCheck{})
	if err != nil {
		return invalidResult, err
	}
	defer resp.Close()

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	wait := 1
	errCh := make(chan error, 2)
	if req.Input != nil {
		wait++
		go func() {
			// Must close write here which will trigger an EOF
			defer resp.CloseWrite()
			_, err := io.Copy(resp.Conn, req.Input)
			errCh <- err
			util.MaybeIOClose(req.Input)
		}()
	}

	go func() {
		_, err = stdcopy.StdCopy(stdout, stderr, resp.Reader)
		errCh <- err
	}()

wait:
	for {
		select {
		case err := <-errCh:
			if err != nil && err != io.EOF {
				return invalidResult, err
			}
			wait--
			if wait == 0 {
				break wait
			}
			// nothing
		case <-ctx.Done():
			// timeout
			return invalidResult, ctx.Err()
		}
	}

	inspect, err2 := e.Client.ContainerExecInspect(ctx, create.ID)
	if err == nil {
		err = err2
	}
	// When exec successfully but with exitCode!=0, I wrap it as an error. This forces developers to handle errors.
	if err == nil && inspect.ExitCode != 0 {
		err = fmt.Errorf("exitcode=[%d] stdout=[%s] stderr=[%s]", inspect.ExitCode, stdout.String(), stderr.String())
	}
	return cri.ExecResult{Cmd: invalidResult.Cmd, ExitCode: inspect.ExitCode, Stdout: stdout, Stderr: stderr}, err
}

func (e *DockerContainerEngine) CopyToContainer(ctx context.Context, c *cri.Container, src, dst string) error {
	// mkdir -p
	if _, err := e.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"mkdir", "-p", filepath.Dir(src)}}); err != nil {
		return err
	}
	return copyToContainerByDockerAPI(e.Client, ctx, c, src, dst)
}

func (e *DockerContainerEngine) CopyFromContainer(ctx context.Context, c *cri.Container, src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return copyFromContainerByDockerAPI(e.Client, ctx, c, src, dst)
}

func (e *DockerContainerEngine) Supports(feature cri.ContainerEngineFeature) bool {
	switch feature {
	case cri.ContainerEngineFeatureCopy:
		// TODO Whether copy is supported depends on the lower-level container runtime rather than the higher-level container runtime.
		return !e.isPouch
	default:
		return false
	}
}

func (e *DockerContainerEngine) ExecAsync(ctx context.Context, c *cri.Container, req cri.ExecRequest) (cri.ExecAsyncResult, error) {
	resultCh := make(chan cri.ExecAsyncResultCode)
	invalidResult := cri.ExecAsyncResult{Cmd: strings.Join(req.Cmd, " "), Result: resultCh}
	create, err := e.Client.ContainerExecCreate(ctx, c.Id, types.ExecConfig{
		User:         req.User,
		Privileged:   false,
		Tty:          false,
		AttachStdin:  req.Input != nil,
		AttachStderr: !req.NoStdErr,
		AttachStdout: true,
		Detach:       false,
		DetachKeys:   "",
		Env:          req.Env,
		WorkingDir:   req.WorkingDir,
		Cmd:          req.Cmd,
	})
	if err != nil {
		return invalidResult, err
	}

	resp, err := e.Client.ContainerExecAttach(ctx, create.ID, types.ExecStartCheck{})
	if err != nil {
		return invalidResult, err
	}

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	errCh := make(chan error, 2)
	wait := 1
	if req.Input != nil {
		wait++
		go func() {
			// Must close write here which will trigger an EOF
			defer resp.CloseWrite()
			_, err := io.Copy(resp.Conn, req.Input)
			errCh <- err
			util.MaybeIOClose(req.Input)
		}()
	}

	go func() {
		_, err := stdcopy.StdCopy(stdoutW, stderrW, resp.Reader)
		stdoutW.Close()
		stderrW.Close()
		errCh <- err
	}()

	go func() {

		for {
			select {
			case err := <-errCh:
				if err != nil && err != io.EOF {
					resp.Close()
					resultCh <- cri.ExecAsyncResultCode{
						Code: -1,
						Err:  err,
					}
					return
				}

				wait--
				if wait == 0 {
					// nothing
					inspect, err2 := e.Client.ContainerExecInspect(ctx, create.ID)
					if err == nil {
						err = err2
					}
					// When exec successfully but with exitCode!=0, I wrap it as an error. This forces developers to handle errors.
					if err == nil && inspect.ExitCode != 0 {
						err = fmt.Errorf("exitcode=[%d]", inspect.ExitCode)
					}
					resultCh <- cri.ExecAsyncResultCode{
						Code: inspect.ExitCode,
						Err:  err,
					}
					resp.Close()
					return
				}
			case <-ctx.Done():
				resp.Close()
				resultCh <- cri.ExecAsyncResultCode{
					Code: -1,
					Err:  ctx.Err(),
				}
				return
			}
		}
	}()

	return cri.ExecAsyncResult{Cmd: invalidResult.Cmd, Result: resultCh, Stdout: stdoutR, Stderr: stderrR}, nil
}
