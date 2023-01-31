package handlers

import (
	"encoding/json"
	"github.com/TRaaSStack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/cmds/previewlog"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"os"
)

func previewLogHandler(action string, resp *model.Resp) error {
	resp.Success = true
	req := &pb.PreviewFileRequest{}
	err := json.NewDecoder(os.Stdin).Decode(req)
	if err != nil {
		return err
	}
	resp0 := &pb.PreviewFileResponse{}
	err = previewlog.PreviewFile(req, resp0)
	if err != nil {
		return err
	}
	resp.Data = resp0
	return nil
}
