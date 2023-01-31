package bistream

import (
	"context"
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/TRaaSStack/holoinsight-agent/pkg/bistream/cmds"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	commonpb "github.com/TRaaSStack/holoinsight-agent/pkg/server/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/bistream"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var hr *registry.HandlerRegistry

func init() {
	hr = registry.NewHandlerRegistry()
	// 注册各种handler
	hr.RegisterRpc(biztypes.BizTypeEcho, func(i int32, bytes []byte) (int32, interface{}, error) {
		// logger.Infoz("echo")
		return i, bytes, nil
		//return 0, nil, errors.New("unsupported")
	})

	hr.RegisterRpc(biztypes.BizTypeListDir, cmds.ListFiles)
	hr.RegisterRpc(biztypes.BizTypePreviewFile, cmds.PreviewFile)
	hr.RegisterRpc(biztypes.BizTypeSplitLog, cmds.SplitLog)
	hr.RegisterRpc(biztypes.BizTypeInspect, cmds.Inspect)
	hr.RegisterRpc(biztypes.BizTypeMatchFiles, cmds.MatchFiles)
	hr.RegisterRpc(biztypes.BizTypeDryRun, cmds.DryRun)
	hr.RegisterRpc(biztypes.BizTypeHttpProxy, cmds.HttpProxy)
	hr.RegisterRpc(biztypes.BizCheckTask, cmds.CheckTask)

	hr.RegisterHandshake(HandshakeHandlerImpl)
}

func GetBiStreamHandlerRegistry() *registry.HandlerRegistry {
	return hr
}

func HandshakeHandlerImpl(ctx context.Context, c pb.RegistryServiceForAgent_BiStreamsClient, version int64) error {
	sendErr := make(chan error, 1)
	go func() {
		req := &pb.BiStreamClientHandshakeRequest{
			AgentId: agentmeta.GetAgentId(),
			Version: version,
		}
		b, err := proto.Marshal(req)
		if err != nil {
			sendErr <- err
			return
		}
		err = c.Send(&commonpb.GenericRpcCommand{
			RpcType: bistream.RpcTypeClientHandShake,
			ReqId:   0,
			BizType: 0,
			Data:    b,
		})
		sendErr <- err
	}()

	select {
	case err := <-sendErr:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	recvErr := make(chan error, 1)
	go func() {
		resp, err := c.Recv()
		if err != nil {
			recvErr <- err
			return
		}
		if resp.RpcType == bistream.RpcTypeServerHandShake {
			biStreamResp := pb.BiStreamClientHandshakeResponse{}
			err := proto.Unmarshal(resp.Data, &biStreamResp)
			if err != nil {
				recvErr <- err
				return
			}
			if biStreamResp.Header != nil && biStreamResp.Header.Code != 0 {
				recvErr <- fmt.Errorf("server handshake error %+v", resp)
				return
			}
			logger.Debugz("recv server handshake", zap.Stringer("resp", resp))
			recvErr <- nil
		} else {
			recvErr <- fmt.Errorf("expect RpcTypeServerHandShake but get %s", resp.String())
		}
	}()

	select {
	case err := <-recvErr:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
