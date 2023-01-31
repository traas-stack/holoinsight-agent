package cmds

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/biztypes"
	commonpb "github.com/TRaaSStack/holoinsight-agent/pkg/server/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"google.golang.org/protobuf/proto"
	"regexp"
)

func SplitLog(_ int32, reqBytes []byte) (int32, interface{}, error) {
	resp := &pb.SplitLogResponse{
		Header: &commonpb.CommonResponseHeader{},
	}
	err := splitLog0(reqBytes, resp)
	fillError(resp.Header, err)
	return biztypes.BizTypeSplitLogResp, resp, nil
}

func splitLog0(reqBytes []byte, resp *pb.SplitLogResponse) error {
	req := pb.SplitLogRequest{}
	err := proto.Unmarshal(reqBytes, &req)
	if err != nil {
		return err
	}

	r, err := regexp.Compile(req.Regexp)
	if err != nil {
		return err
	}

	g := r.FindStringSubmatch(req.Content)
	if g == nil {
		resp.Header.Code = 1
		resp.Header.Message = "not match"
	} else {
		result := make(map[string]string)
		for i, s := range r.SubexpNames() {
			if i == 0 {
				continue
			}
			result[s] = g[i]
		}
		resp.Result = result
	}
	return nil
}
