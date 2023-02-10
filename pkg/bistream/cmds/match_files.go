package cmds

import (
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/rpccodes"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/filematch"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"google.golang.org/protobuf/proto"
)

func MatchFiles(_ int32, bytes []byte) (int32, interface{}, error) {
	resp := &pb.MatchFilesResponse{
		Header: &commonpb.CommonResponseHeader{},
	}
	err := matchFiles0(bytes, resp)
	fillError(resp.Header, err)
	return biztypes.BizTypeMatchFilesResp, resp, nil
}

func matchFiles0(bytes []byte, resp *pb.MatchFilesResponse) error {
	req := pb.MatchFilesRequest{}
	err := proto.Unmarshal(bytes, &req)
	if err != nil {
		return err
	}

	var fm filematch.FileMatcher

	switch req.Type {
	case filematch.TypePath:
		fm = filematch.NewAbsFileMatcher(req.Pattern)
	case filematch.TypeGlob:
		fm, err = filematch.NewGlobFileMatcher(req.Pattern)
		if err != nil {
			return err
		}
	case filematch.TypeRegexp:
		fm, err = filematch.NewRegexpFileMatcher(req.Dir, req.Pattern, int(req.MaxVisited), int(req.MaxMatched))
		if err != nil {
			return err
		}
	case filematch.TypeFormat:
		fm = filematch.NewFormatFileMatcher(req.Pattern)
	default:
		resp.Header.Code = rpccodes.ILLEGAL_ARGUMENTS
	}

	if fm != nil {
		fatFiles, visited, _ := fm.Find()
		resp.Visited = int32(visited)
		resp.Files = filematch.GetPaths(fatFiles)
	}

	return nil
	// type: path/format/glob/regexp
	// pattern string 其含义取决于type
	// dir string, 当type==regexp时有效
}
