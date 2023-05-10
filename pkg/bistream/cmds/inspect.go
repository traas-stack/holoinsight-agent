/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cmds

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/cmds/inspect"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"google.golang.org/protobuf/proto"
)

func Inspect(_ int32, bytes []byte) (int32, interface{}, error) {
	resp := &pb.InspectResponse{
		Header: &commonpb.CommonResponseHeader{},
	}
	err := inspect0(bytes, resp)
	fillError(resp.Header, err)
	return biztypes.BizTypeInspectResp, resp, err
}

func inspect0(reqBytes []byte, resp *pb.InspectResponse) error {
	req := &pb.InspectRequest{}
	err := proto.Unmarshal(reqBytes, req)
	if err != nil {
		return err
	}

	if crii, container, err := getPodContainer(req.Header); err != nil {
		return err
	} else if container != nil {
		return runInContainer(resp, func(ctx context.Context) (cri.ExecResult, error) {
			env := []string{"AGENT_INFO=" + util.ToJsonString(inspect.CreateAgentInfo())}
			return crii.Exec(ctx, container, cri.ExecRequest{
				Cmd: []string{core.HelperToolPath, "inspect"},
				Env: env,
			})
		})
	}

	return inspect.Inspect(nil, resp)
}
