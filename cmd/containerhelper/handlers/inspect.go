package handlers

import (
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/cmds/inspect"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
)

func inspectHandler(action string, resp *model.Resp) error {
	req := &pb.InspectRequest{}
	//err := json.NewDecoder(os.Stdin).Decode(req)
	//if err != nil {
	//	return err
	//}
	resp0 := &pb.InspectResponse{}
	err := inspect.Inspect(req, resp0)
	if err != nil {
		return err
	}
	resp.Success = true
	resp.Data = resp0
	return nil
}
