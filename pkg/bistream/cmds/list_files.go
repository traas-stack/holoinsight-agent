package cmds

import (
	"context"
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/cmds/listfiles"
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	commonpb "github.com/TRaaSStack/holoinsight-agent/pkg/server/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"google.golang.org/protobuf/proto"
)

func ListFiles(_ int32, bytes []byte) (int32, interface{}, error) {
	resp := &pb.ListFilesResponse{
		Header: &commonpb.CommonResponseHeader{},
	}
	err := listFiles0(bytes, resp)
	fillError(resp.Header, err)
	return biztypes.BizTypeListDirResp, resp, nil
}

func listFiles0(bs []byte, resp *pb.ListFilesResponse) error {
	req := &pb.ListFilesRequest{}
	err := proto.Unmarshal(bs, req)
	if err != nil {
		return err
	}

	if crii, container, err := getPodContainer(req.Header); err != nil {
		return err
	} else if container != nil {
		input, err := util.ToJsonBufferE(req)
		if err != nil {
			return err
		}
		return runInContainer(resp, func(ctx context.Context) (cri.ExecResult, error) {
			return crii.NsEnterHelperExec(ctx, container, []string{"listFiles"}, nil, "", input)
		})
	}

	return listfiles.ListFiles(req, resp)
}
