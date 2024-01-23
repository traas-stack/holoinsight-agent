/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cmds

import (
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/cmds/previewlog"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/dockerutils"
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
		var hostPath string
		isDockerJsonLog := req.Path == dockerutils.DockerJsonLogFile
		if isDockerJsonLog {
			hostPath = container.LogPath
		} else {
			hostPath, err = cri.TransferToHostPathForContainer(container, req.Path, true)
			if err != nil {
				return err
			}
		}

		req2 := &pb.PreviewFileRequest{}
		_ = proto.Unmarshal(reqBytes, req2)
		req2.Path = hostPath

		resp.Timezone = container.GetTzName()
		err = previewlog.PreviewFile(req2, resp)
		if err == nil && isDockerJsonLog {
			for i := range resp.Content {
				if dl, err := dockerutils.DecodeJsonLog(resp.Content[i]); err == nil {
					resp.Content[i] = dl.Log
				}
			}
		}
		return trimErrorPathInfo(err, hostPath, req.Path)
	}

	return previewlog.PreviewFile(req, resp)
}
