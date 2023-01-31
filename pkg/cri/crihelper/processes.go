package crihelper

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	"github.com/TRaaSStack/holoinsight-agent/pkg/ioc"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"context"
	"encoding/json"
	"github.com/spf13/cast"
)

func HelperProcesses(ctx context.Context, pod *cri.Pod, pid int) (*ProcessInfo, error) {
	er, err := ioc.Crii.NsEnterHelperExec(ctx, pod.MainBiz(), []string{"processes", cast.ToString(pid)}, nil, "", nil)
	stdout := er.Stdout.String()
	stderr := er.Stderr.String()

	if logger.DebugEnabled {
		logger.Debugf("[jvm] exitcode=[%d] stdout=[%s] stderr=[%s] %+v", er.ExitCode, stdout, stderr, err)
	}

	if err != nil {
		return nil, err
	}

	resp := struct {
		Success bool         `json:"success"`
		Message bool         `json:"message"`
		Data    *ProcessInfo `json:"data"`
	}{}
	err = json.Unmarshal([]byte(stdout), &resp)
	return resp.Data, err
}
