package meta

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	dockersdk "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"io"
)

func execSync(docker *dockersdk.Client, ctx context.Context, c *cri.Container, cmd []string, env []string, workingDir string, input io.Reader) (cri.ExecResult, error) {
	create, err := docker.ContainerExecCreate(ctx, c.Id, types.ExecConfig{
		// TODO exec 进去后不一定有权限
		// 可以用root登录!
		User: "root",
		// TODO 这个参数有用?
		Privileged:   false,
		Tty:          false,
		AttachStdin:  input != nil,
		AttachStderr: true,
		AttachStdout: true,
		Detach:       false,
		DetachKeys:   "",
		Env:          env,
		WorkingDir:   workingDir,
		Cmd:          cmd,
	})
	if err != nil {
		return cri.ExecResult{ExitCode: -1}, err
	}
	resp, err := docker.ContainerExecAttach(ctx, create.ID, types.ExecStartCheck{})
	if err != nil {
		return cri.ExecResult{ExitCode: -1}, err
	}
	defer resp.Close()

	copyDone := make(chan struct{}, 1)

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	if input != nil {
		go func() {
			io.Copy(resp.Conn, input)
		}()
	}

	go func() {
		_, err = stdcopy.StdCopy(stdout, stderr, resp.Reader)
		copyDone <- struct{}{}
	}()
	select {
	case <-copyDone:
		// nothing
	case <-ctx.Done():
		// timeout
		return cri.ExecResult{ExitCode: -1}, err
	}

	inspect, err2 := docker.ContainerExecInspect(ctx, create.ID)
	if err == nil {
		err = err2
	}
	if err == nil && inspect.ExitCode != 0 {
		err = fmt.Errorf("exitcode=[%d] stdout=[%s] stderr=[%s]", inspect.ExitCode, stdout.String(), stderr.String())
	}
	return cri.ExecResult{ExitCode: inspect.ExitCode, Stdout: stdout, Stderr: stderr}, err
}
