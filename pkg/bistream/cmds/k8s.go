package cmds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/core"
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	"github.com/TRaaSStack/holoinsight-agent/pkg/ioc"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	commonpb "github.com/TRaaSStack/holoinsight-agent/pkg/server/pb"
	"go.uber.org/zap"
	"time"
)

type (
	simplePodInfo struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
)

func fillError(h *commonpb.CommonResponseHeader, err error) {
	if err == nil {
		return
	}
	h.Code = 400
	h.Message = err.Error()
}

func getPodContainer(header *commonpb.CommonRequestHeader) (cri.Interface, *cri.Container, error) {
	if header == nil {
		return nil, nil, nil
	}
	podStr := header.Header["pod"]
	if podStr == "" {
		return nil, nil, nil
	}
	var pi simplePodInfo
	if err := json.Unmarshal([]byte(podStr), &pi); err != nil {
		logger.Errorz("getPodContainer parse pod info error", zap.Error(err))
		return nil, nil, errors.New("fail to parse pod info")
	}

	if pod, ok := ioc.Crii.GetPod(pi.Namespace, pi.Name); ok {
		mainBiz := pod.MainBiz()
		if mainBiz == nil {
			return nil, nil, errors.New("no main biz container")
		}
		return ioc.Crii, mainBiz, nil
	} else {
		return nil, nil, errors.New("no such pod")
	}

	return nil, nil, nil
}

func runInContainer(resp interface{}, callHelper func(ctx context.Context) (cri.ExecResult, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//env := []string{"AGENT_INFO=" + util.ToJsonString(inspect.CreateAgentInfo())}
	// er, err := crii.NsEnterHelperExec(ctx, container, []string{"inspect"}, env, "", nil)

	er, err := callHelper(ctx)
	if err != nil {
		return err
	}

	type tempResp struct {
		core.HelperBaseResp
		// Resp *pb.InspectResponse `json:"data"`
		Resp interface{} `json:"data"`
	}
	temp := &tempResp{Resp: resp}
	err = json.NewDecoder(er.Stdout).Decode(temp)
	if err != nil {
		return err
	}
	if !temp.Success {
		return fmt.Errorf("inspect error %s", temp.Message)
	}
	return nil
}
