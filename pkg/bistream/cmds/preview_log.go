package cmds

import (
	"context"
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/cmds/previewlog"
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	commonpb "github.com/TRaaSStack/holoinsight-agent/pkg/server/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"google.golang.org/protobuf/proto"
)

const (
	maxBytes = 1024 * 1024
)

func PreviewFile(_ int32, reqBytes []byte) (int32, interface{}, error) {
	resp := &pb.PreviewFileResponse{
		Header: &commonpb.CommonResponseHeader{},
		// TODO 猜测编码
		Charset: "UTF-8",
	}
	err := previewFile0(reqBytes, resp)
	fillError(resp.Header, err)
	return biztypes.BizTypePreviewFileResp, resp, nil
}

func previewFile0(reqBytes []byte, resp *pb.PreviewFileResponse) error {
	req := &pb.PreviewFileRequest{}
	err := proto.Unmarshal(reqBytes, req)
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
			return crii.NsEnterHelperExec(ctx, container, []string{"previewLog"}, nil, "", input)
		})
	}

	return previewlog.PreviewFile(req, resp)
}
