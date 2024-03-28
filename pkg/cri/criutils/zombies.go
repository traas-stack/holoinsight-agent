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

func CountZombies(i cri.Interface, ctx context.Context, c *cri.Container) (int, error) {
	r, err := i.Exec(ctx, c, cri.ExecRequest{
		Cmd:                  []string{core.HelperToolPath, "countZombies"},
		NoWrapCmdWithTimeout: true,
	})
	if err != nil {
		return 0, err
	}
	resp := &model.Resp{}
	if err := json.NewDecoder(r.Stdout).Decode(resp); err != nil {
		return 0, err
	}
	return cast.ToIntE(resp.Data)
}
