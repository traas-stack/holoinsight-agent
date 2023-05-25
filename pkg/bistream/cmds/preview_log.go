/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cmds

import (
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/cmds/previewlog"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"google.golang.org/protobuf/proto"
)

const (
	maxBytes = 1024 * 1024
)

func PreviewFile(_ int32, reqBytes []byte) (int32, interface{}, error) {
	resp := &pb.PreviewFileResponse{
		Header: &commonpb.CommonResponseHeader{},
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

	if _, container, err := getPodContainer(req.Header); err != nil {
		return err
	} else if container != nil {
		hostPath, err := cri.TransferToHostPathForContainer(container, req.Path, true)
		if err != nil {
			return err
		}

		req2 := &pb.PreviewFileRequest{}
		_ = proto.Unmarshal(reqBytes, req2)
		req2.Path = hostPath

		return trimErrorPathInfo(previewlog.PreviewFile(req2, resp), hostPath, req.Path)
	}

	return previewlog.PreviewFile(req, resp)
}
