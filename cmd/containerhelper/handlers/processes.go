/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"os"
)

func processesHandler(action string, resp *model.Resp) error {
	pid := cast.ToInt32(os.Args[2])
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}

	data := map[string]interface{}{}
	resp.Data = data
	data["user"], _ = p.Username()
	data["name"], _ = p.Name()
	data["cmdlineSlice"], _ = p.CmdlineSlice()
	data["tgid"], _ = p.Tgid()
	resp.Success = true
	return nil
}
