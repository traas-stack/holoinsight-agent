/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cmds

import (
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/dryrun/event"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"google.golang.org/protobuf/proto"
)

func DryRun(_ int32, bytes []byte) (int32, interface{}, error) {
	resp := &pb.DryRunResponse{
		Header: &commonpb.CommonResponseHeader{},
	}
	err := dryRun0(bytes, resp)
	fillError(resp.Header, err)
	return biztypes.BizTypeDryRunResp, resp, nil
}

func dryRun0(bytes []byte, resp *pb.DryRunResponse) error {
	pbReq := &pb.DryRunRequest{}
	err := proto.Unmarshal(bytes, pbReq)
	if err != nil {
		return err
	}

	dreq := &executor.DryRunRequest{
		Task: &collecttask.CollectTask{
			Key:     "DRYRUN",
			Version: "DRYRUN",
			Config: &collecttask.CollectConfig{
				Key:     "DRYRUN",
				Type:    "SQLTASK",
				Version: "DRYRUN",
				Content: []byte(pbReq.Config.Temp.Json),
			},
			Target: &collecttask.CollectTarget{},
		},
		Input: &executor.Input{
			Type: "plain",
			Plain: &executor.Input_Plain{
				Lines:    pbReq.Input.Plain.Lines,
				Timezone: pbReq.Input.Plain.Timezone,
			},
			Read: nil,
		},
	}

	executor, err := executor.NewDryRunExecutor(dreq)

	if err != nil {
		return err
	}

	dresp := executor.Run()

	resp.Event = convertEventToPbDfs(dresp.Event)

	//dresp := executor.Run()
	// dresp.Event
	return nil
}

func convertEventToPbDfs(event *event.Event) *pb.DryRunResponse_Event {
	pbe := &pb.DryRunResponse_Event{
		Title:  event.Title,
		Params: make(map[string]string),
	}
	for k, v := range event.Params {
		pbe.Params[k] = util.ToJsonString(v)
	}
	for _, msg := range event.Messages {
		pbe.Message = append(pbe.Message, &pb.DryRunResponse_Message{
			Level:   msg.Level,
			Content: msg.Content,
		})
	}
	for _, child := range event.Children {
		pbe.Children = append(pbe.Children, convertEventToPbDfs(child))
	}
	return pbe
}
