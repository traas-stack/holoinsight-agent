/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cmds

import (
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"google.golang.org/protobuf/proto"
)

func CheckTask(_ int32, bytes []byte) (int32, interface{}, error) {
	resp := &pb.CheckConfigTaskDistributionResponse{
		Header: &commonpb.CommonResponseHeader{},
	}
	err := checkTask0(bytes, resp)
	fillError(resp.Header, err)
	return biztypes.BizCheckTaskResp, resp, err
}

func checkTask0(bytes []byte, resp *pb.CheckConfigTaskDistributionResponse) error {
	pbReq := &pb.CheckConfigTaskDistributionRequest{}
	err := proto.Unmarshal(bytes, pbReq)
	if err != nil {
		return err
	}

	for _, task := range pbReq.SubTasks {
		status := int32(ioc.CollectTaskManager.CheckTask(task.ConfigKey, task.ConfigVersion, task.TargetKey, task.TargetVersion))
		resp.Status = append(resp.Status, status)
	}
	return nil
}
