/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

import (
	"context"
	"encoding/json"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
)

// GetProcessInfo returns process info of pid in target container
func GetProcessInfo(ctx context.Context, i cri.Interface, c *cri.Container, pid int) (*ProcessInfo, error) {
	er, err := i.Exec(ctx, c, cri.ExecRequest{
		Cmd: []string{core.HelperToolPath, "processes", cast.ToString(pid)},
	})
	if err != nil {
		return nil, err
	}

	stdout := er.Stdout.String()
	processesResp := struct {
		model.RespBase
		Data *ProcessInfo `json:"data"`
	}{}
	err = json.Unmarshal([]byte(stdout), &processesResp)
	return processesResp.Data, err
}
