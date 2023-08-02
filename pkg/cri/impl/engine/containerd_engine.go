/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package engine

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/cio"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"io"
	"os"
	"path/filepath"
	"strings"

	containerstore "github.com/containerd/containerd/pkg/cri/store/container"
	sandboxstore "github.com/containerd/containerd/pkg/cri/store/sandbox"
	_ "github.com/containerd/containerd/runtime"
	"github.com/containerd/typeurl"
	"github.com/google/cadvisor/container/containerd/namespaces"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
)

const (
	IDLength = 64
	k8sNs    = "k8s.io"
)

type (
	// ContainerdContainerEngine Containerd container engine
	ContainerdContainerEngine struct {
		Client  *containerd.Client
		fifoDir string
	}
)

var (
	// Make sure *ContainerdContainerEngine impl cri.ContainerEngine
	_ cri.ContainerEngine = &ContainerdContainerEngine{}
)

func init() {
	typeurl.Register(&containerstore.Metadata{},
		"github.com/containerd/cri/pkg/store/container", "Metadata")
	typeurl.Register(&sandboxstore.Metadata{},
		"github.com/containerd/cri/pkg/store/sandbox", "Metadata")
}

func (e *ContainerdContainerEngine) Init() error {
	// When execute 'ctr exec' using containerd sdk, it requires:
	// 1. fifo dir is visible to current process (HoloInsight-Agent)

	// Because HoloInsight-Agent daemonset runs in its own MNT namespace, the filesystems of it and host are different.
	// But we mount hostPath `/usr/local/holoinsight/agent/data` to containerPath `/usr/local/holoinsight/agent/data`.
	// So only these few directory are the same in two filesystems.
	// So we can set the FIFODir for 'containerd exec' to '/usr/local/holoinsight/agent/data/containrd-fifo'.

	workdir, err := os.Getwd()
	if err != nil {
		return err
	}
	e.fifoDir = filepath.Join(workdir, "data", "containerd-fifo")
	if err := os.MkdirAll(e.fifoDir, 0755); err != nil {
		return err
	}

	//if err := mount.SetTempMountLocation(filepath.Join(workdir, "data", "containerd-mount")); err != nil {
	//	return err
	//}

	// TODO events
	//go func() {
	//
	//	eventCh, errCh := e.Client.EventService().Subscribe(context.Background())
	//	select {
	//	case event := <-eventCh:
	//		logger.Dockerz("[event]", zap.String("engine", e.Type()), zap.Any("event", event))
	//	case err := <-errCh:
	//		logger.Dockerz("[event] error", zap.String("engine", e.Type()), zap.Error(err))
	//	}
	//}()

	return nil
}

func (e *ContainerdContainerEngine) Type() string {
	return "containerd"
}

func wrapK8sCtx(ctx context.Context) context.Context {
	return namespaces.WithNamespace(ctx, k8sNs)
}

func (e *ContainerdContainerEngine) ListAllContainers(ctx context.Context) ([]*cri.EngineSimpleContainer, error) {
	ctx = wrapK8sCtx(ctx)
	containers, err := e.Client.ContainerService().List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*cri.EngineSimpleContainer, len(containers))
	for i := range containers {
		copy := containers[i]
		items[i] = &cri.EngineSimpleContainer{
			ID:     copy.ID,
			Labels: copy.Labels,
			Source: copy,
		}
	}
	return items, nil
}

func (e *ContainerdContainerEngine) GetContainerDetail(ctx context.Context, cid string) (*cri.EngineDetailContainer, error) {
	ctx = wrapK8sCtx(ctx)

	container, err := e.Client.ContainerService().Get(ctx, cid)
	if err != nil {
		return nil, err
	}

	spec := specs.Spec{}
	if err := typeurl.UnmarshalToByTypeURL(container.Spec.GetTypeUrl(), container.Spec.GetValue(), &spec); err != nil {
		return nil, err
	}

	cm := containerstore.Metadata{}
	sm := sandboxstore.Metadata{}

	for key, ext := range container.Extensions {
		switch key {
		case "io.cri-containerd.container.metadata":
			if err = typeurl.UnmarshalToByTypeURL(ext.GetTypeUrl(), ext.GetValue(), &cm); err != nil {
				return nil, err
			}
		case "io.cri-containerd.sandbox.metadata":
			if err = typeurl.UnmarshalToByTypeURL(ext.GetTypeUrl(), ext.GetValue(), &sm); err != nil {
				return nil, err
			}
		}
	}
	detail := &cri.EngineDetailContainer{
		ID:       container.ID,
		Labels:   container.Labels,
		Env:      spec.Process.Env,
		Source:   container,
		Hostname: spec.Hostname,

		IsSandbox: container.Labels["io.cri-containerd.kind"] == "sandbox",

		Mounts:    nil,
		Runtime:   "",
		SandboxId: "",
		// Name formats are different in docker and containerd
		Name: "",

		MergedDir:   "",
		NetworkMode: "",
		State: cri.ContainerState{
			Pid: 0,
		},
	}

	if taskResp, err := e.Client.TaskService().Get(ctx, &tasks.GetRequest{
		ContainerID: container.ID,
	}); err == nil {
		detail.State.Pid = int(taskResp.Process.Pid)
		detail.State.Status = strings.ToLower(taskResp.Process.Status.String())
	} else {
		// no task for container : like container dead in docker
	}

	if detail.IsSandbox {
		detail.Name = sm.Name
	} else {
		detail.Name = cm.Name
		detail.SandboxId = cm.SandboxID
	}

	switch container.Runtime.Name {
	case "io.containerd.runc.v2":
		detail.Runtime = "runc"
	default:
		// TODO other runtime
		detail.Runtime = "unknown"
	}

	if !detail.IsSandbox {
		for _, mount := range cm.Config.Mounts {
			detail.Mounts = append(detail.Mounts, &cri.MountPoint{
				Source:      mount.HostPath,
				Destination: mount.ContainerPath,
				RW:          !mount.Readonly,
			})
		}
	}

	if detail.IsSandbox {
		detail.NetworkMode = "netns:" + sm.NetNSPath
	}
	return detail, nil
}

func (e *ContainerdContainerEngine) Exec(ctx context.Context, c *cri.Container, req cri.ExecRequest) (cri.ExecResult, error) {
	// The code for this function refers to: https://github.com/containerd/containerd/blob/main/cmd/ctr/commands/tasks/exec.go

	ctx = wrapK8sCtx(ctx)

	invalidResult := cri.ExecResult{Cmd: strings.Join(req.Cmd, " "), ExitCode: -1}

	container, err := e.Client.LoadContainer(ctx, c.Id)
	if err != nil {
		return invalidResult, err
	}

	spec, err := container.Spec(ctx)
	if err != nil {
		return invalidResult, err
	}

	// TODO
	// WithUser depends on mounts. This will fail when run inside a container.
	// Unless I manually copy a copy of the code of oci.WithUsername(), add the prefix of /hostfs to the mount directory.

	// container2, err := container.Info(ctx)
	// if err != nil {
	// 	return invalidResult, err
	// }

	// if err := oci.WithUserID(0)(ctx, e.Client, &container2, spec); err != nil {
	// 	 return invalidResult, err
	// }

	// It just so happens that in most cases we execute as root, so simply set this to 0
	// TODO I found there is no root user in prometheus/node-exporter image.
	// This image is very thin!
	spec.Process.User.UID = 0
	spec.Process.User.GID = 0
	spec.Process.User.AdditionalGids = []uint32{}
	spec.Process.User.Umask = nil

	pspec := spec.Process
	pspec.Terminal = false
	pspec.Args = req.Cmd
	if req.WorkingDir != "" {
		pspec.Cwd = req.WorkingDir
	}

	// Append user specified env
	pspec.Env = append(pspec.Env, req.Env...)

	task, err := container.Task(ctx, nil)
	if err != nil {
		return invalidResult, err
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	var ioCreator cio.Creator

	var stdinWrapper *util.ReaderCloserFunc
	if req.Input != nil {
		stdinWrapper = &util.ReaderCloserFunc{
			Reader: req.Input,
		}
		ioCreator = cio.NewCreator(cio.WithStreams(stdinWrapper, stdout, stderr), cio.WithFIFODir(e.fifoDir))
	} else {
		ioCreator = cio.NewCreator(cio.WithStreams(nil, stdout, stderr), cio.WithFIFODir(e.fifoDir))
	}

	// Containerd has a exec id limit with max length = 76
	execID := "exec-" + generateID()
	process, err := task.Exec(ctx, execID, pspec, ioCreator)
	if err != nil {
		return invalidResult, err
	}

	if stdinWrapper != nil {
		stdinWrapper.Closer = func() { process.CloseIO(ctx, containerd.WithStdinCloser) }
	}

	defer process.Delete(ctx)

	statusC, err := process.Wait(ctx)
	if err != nil {
		return invalidResult, err
	}

	if err := process.Start(ctx); err != nil {
		return invalidResult, err
	}

	var status containerd.ExitStatus
	select {
	case <-ctx.Done():
		// timeout
		return invalidResult, ctx.Err()
	case status = <-statusC:
	}

	code, _, err := status.Result()

	// When exec successfully but with exitCode!=0, I wrap it as an error. This forces developers to handle errors.
	if err == nil && code != 0 {
		err = fmt.Errorf("exitcode=[%d] stdout=[%s] stderr=[%s]", code, stdout.String(), stderr.String())
	}
	return cri.ExecResult{Cmd: invalidResult.Cmd, ExitCode: int(code), Stdout: stdout, Stderr: stderr}, err

}

func (e *ContainerdContainerEngine) CopyToContainer(ctx context.Context, c *cri.Container, src, dst string) error {
	return errors.New("CopyToContainer unsupported")
}

func (e *ContainerdContainerEngine) CopyFromContainer(ctx context.Context, c *cri.Container, src, dst string) error {
	return errors.New("CopyFromContainer unsupported")
}

func (e *ContainerdContainerEngine) Supports(feature cri.ContainerEngineFeature) bool {
	switch feature {
	case cri.ContainerEngineFeatureCopy:
		return false
	default:
		return false
	}
}

// generateID is copy from nerdctl idgen.go
// Generate id with length = 64
func generateID() string {
	bytesLength := IDLength / 2
	b := make([]byte, bytesLength)
	n, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	if n != bytesLength {
		panic(fmt.Errorf("expected %d bytes, got %d bytes", bytesLength, n))
	}
	return hex.EncodeToString(b)
}

func (e *ContainerdContainerEngine) ExecAsync(ctx context.Context, c *cri.Container, req cri.ExecRequest) (cri.ExecAsyncResult, error) {
	// The code for this function refers to: https://github.com/containerd/containerd/blob/main/cmd/ctr/commands/tasks/exec.go

	ctx = wrapK8sCtx(ctx)

	result := make(chan cri.ExecAsyncResultCode)
	invalidResult := cri.ExecAsyncResult{Cmd: strings.Join(req.Cmd, " "), Result: result}

	container, err := e.Client.LoadContainer(ctx, c.Id)
	if err != nil {
		return invalidResult, err
	}

	spec, err := container.Spec(ctx)
	if err != nil {
		return invalidResult, err
	}

	// TODO
	// WithUser depends on mounts. This will fail when run inside a container.
	// Unless I manually copy a copy of the code of oci.WithUsername(), add the prefix of /hostfs to the mount directory.

	// container2, err := container.Info(ctx)
	// if err != nil {
	// 	return invalidResult, err
	// }

	// if err := oci.WithUserID(0)(ctx, e.Client, &container2, spec); err != nil {
	// 	 return invalidResult, err
	// }

	// It just so happens that in most cases we execute as root, so simply set this to 0
	// TODO I found there is no root user in prometheus/node-exporter image.
	// This image is very thin!
	spec.Process.User.UID = 0
	spec.Process.User.GID = 0
	spec.Process.User.AdditionalGids = []uint32{}
	spec.Process.User.Umask = nil

	pspec := spec.Process
	pspec.Terminal = false
	pspec.Args = req.Cmd
	if req.WorkingDir != "" {
		pspec.Cwd = req.WorkingDir
	}

	// Append user specified env
	pspec.Env = append(pspec.Env, req.Env...)

	task, err := container.Task(ctx, nil)
	if err != nil {
		return invalidResult, err
	}

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	var ioCreator cio.Creator

	var stdinWrapper *util.ReaderCloserFunc
	if req.Input != nil {
		stdinWrapper = &util.ReaderCloserFunc{
			Reader: req.Input,
		}
		ioCreator = cio.NewCreator(cio.WithStreams(stdinWrapper, stdoutW, stderrW), cio.WithFIFODir(e.fifoDir))
	} else {
		ioCreator = cio.NewCreator(cio.WithStreams(nil, stdoutW, stderrW), cio.WithFIFODir(e.fifoDir))
	}

	// Containerd has a exec id limit with max length = 76
	execID := "exec-" + generateID()
	process, err := task.Exec(ctx, execID, pspec, ioCreator)
	if err != nil {
		return invalidResult, err
	}

	if stdinWrapper != nil {
		stdinWrapper.Closer = func() { process.CloseIO(ctx, containerd.WithStdinCloser) }
	}

	// defer process.Delete(ctx)

	statusC, err := process.Wait(ctx)
	if err != nil {
		return invalidResult, err
	}

	if err := process.Start(ctx); err != nil {
		return invalidResult, err
	}

	go func() {
		defer process.Delete(ctx)

		var status containerd.ExitStatus
		select {
		case <-ctx.Done():
			// timeout
			result <- cri.ExecAsyncResultCode{
				Code: -1,
				Err:  ctx.Err(),
			}
		case status = <-statusC:
			code, _, err := status.Result()

			// When exec successfully but with exitCode!=0, I wrap it as an error. This forces developers to handle errors.
			if err == nil && code != 0 {
				err = fmt.Errorf("exitcode=[%d]", code)
			}
			result <- cri.ExecAsyncResultCode{
				Code: int(code),
				Err:  err,
			}
		}
	}()

	return cri.ExecAsyncResult{Cmd: invalidResult.Cmd, Result: result, Stdout: stdoutR, Stderr: stderrR}, err
}
