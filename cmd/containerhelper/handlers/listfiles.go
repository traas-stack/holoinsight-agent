package handlers

import (
	"encoding/json"
	"github.com/TRaaSStack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/cmds/listfiles"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
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
