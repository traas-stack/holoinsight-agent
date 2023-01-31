package meta

import (
	"bytes"
	"context"
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	"io"
	"os/exec"
)

var defaultNsEnterTypes = []cri.NsEnterType{
	cri.NsEnter_MNT,
	cri.NsEnter_NET,
	cri.NsEnter_UTS,
}

func execNsEnter(ctx context.Context, hostfs string, nsEnterTypes []cri.NsEnterType, pid int, cmd, env []string, workingDir string, input io.Reader) (cri.ExecResult, error) {
	if len(nsEnterTypes) == 0 {
		nsEnterTypes = defaultNsEnterTypes
	}
	var args []string

	// -m : mount
	// -n : network
	// -p : pid
	// -u : enter UTS namespace (hostname etc)
	// -U : user
	// args = append(args, "-m", "-n", "-p")

	// TODO 有些挂不上去 会引起失败
	// args = append(args, fmt.Sprintf("--user=/hostfs/proc/%d/ns/user", c.Pid))
	// args = append(args, fmt.Sprintf("--pid=/hostfs/proc/%d/ns/pid", c.Pid))

	for _, enterType := range nsEnterTypes {
		switch enterType {
		case cri.NsEnter_MNT:
			args = append(args, fmt.Sprintf("--mount=%s/proc/%d/ns/mnt", hostfs, pid))
		case cri.NsEnter_NET:
			args = append(args, fmt.Sprintf("--net=%s/proc/%d/ns/net", hostfs, pid))
		case cri.NsEnter_UTS:
			args = append(args, fmt.Sprintf("--uts=%s/proc/%d/ns/uts", hostfs, pid))
		}
	}

	args = append(args, cmd...)

	execCmd := exec.CommandContext(ctx, "nsenter", args...)
	execCmd.Env = env
	execCmd.Dir = workingDir

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	execCmd.Stdin = input
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr

	runDone := make(chan error, 1)
	go func() {
		runDone <- execCmd.Run()
	}()

	select {
	case <-ctx.Done():
		// 超时
		return cri.ExecResult{Cmd: execCmd.String(), ExitCode: -1}, ctx.Err()
	case err := <-runDone:
		return cri.ExecResult{Cmd: execCmd.String(), ExitCode: execCmd.ProcessState.ExitCode(), Stdout: stdout, Stderr: stderr}, err
	}
}
