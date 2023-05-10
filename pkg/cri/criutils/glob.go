/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

import (
	"context"
	"encoding/json"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
)

// Glob runs glob commands inside the container.
func Glob(ctx context.Context, i cri.Interface, c *cri.Container, pattern string) ([]string, error) {
	execResult, err := i.Exec(ctx, c, cri.ExecRequest{
		Cmd: []string{
			core.HelperToolPath, "glob", pattern,
		},
	})

	if err != nil {
		return nil, err
	}

	globResults := struct {
		model.RespBase
		Data []string `json:"data"`
	}{}
	err = json.NewDecoder(execResult.Stdout).Decode(&globResults)
	if err != nil {
		return nil, err
	}
	return globResults.Data, nil
}
