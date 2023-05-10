/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cmds

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/cmds/previewlog"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
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
			return crii.Exec(ctx, container, cri.ExecRequest{
				Cmd:   []string{core.HelperToolPath, "previewLog"},
				Input: input,
			})
		})
	}

	return previewlog.PreviewFile(req, resp)
}
