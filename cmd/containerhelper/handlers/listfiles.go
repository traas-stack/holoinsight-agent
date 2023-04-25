/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"encoding/json"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/cmds/listfiles"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"os"
)

func listFilesHandlers(action string, resp *model.Resp) error {
	req := &pb.ListFilesRequest{}
	err := json.NewDecoder(os.Stdin).Decode(req)
	if err != nil {
		return err
	}
	resp0 := &pb.ListFilesResponse{}
	err = listfiles.ListFiles(req, resp0)
	if err != nil {
		return err
	}
	resp.Success = true
	resp.Data = resp0
	return nil
}
