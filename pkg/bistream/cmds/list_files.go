/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cmds

import (
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/cmds/listfiles"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
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

	if _, container, err := getPodContainer(req.Header); err != nil {
		return err
	} else if container != nil {
		hostPath, err := cri.TransferToHostPathForContainer(container, req.Name, false)
		if err != nil {
			return err
		}

		req2 := &pb.ListFilesRequest{}
		_ = proto.Unmarshal(bs, req2)
		req2.Name = hostPath
		resp2 := &pb.ListFilesResponse{}
		if err := listfiles.ListFiles(req2, resp2); err != nil {
			return err
		}
		// rebase to container dir
		newRoot, err := listfiles.Rebase(resp2.Nodes, hostPath, req.Name)
		if err != nil {
			return err
		}
		resp.Nodes = []*commonpb.FileNode{newRoot}
		return nil
	}

	return listfiles.ListFiles(req, resp)
}
