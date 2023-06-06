/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package registry

// 这个包提供与Registry交互的能力, 但需要有另外一个组件来调用它, 比如注册agent/心跳/配置同步/双向流

import (
	"bytes"
	"context"
	"crypto/tls"
	"github.com/golang/protobuf/proto"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	pb2 "github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials/insecure"
	"strings"
	"sync"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/util/eventcollector"

	"github.com/traas-stack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	delayCloseDuration = 2 * time.Minute
	// 每 10min 重连一次 registry, 使得服务端连接打散
	reconnectInterval          = 10 * time.Minute
	syncControlConfigsInterval = time.Minute
	pingTimeout                = 3 * time.Second
)

var (
// 默认开启gzip 节省网络流量
// 我们在云上会更多遇到跨vpc, 此时slb按流量收费
)

type (
	AgentInfo struct {
		AgentId      string
		App          string
		AgentVersion string
		Ip           string
		Hostname     string
		OS           string
		Arch         string
		Labels       map[string]string
		K8s          *pb2.AgentK8SInfo
		Mode         string
		Workspace    string
	}
	// registry 相关的配置
	Config struct {
		// grpc 的 ip:port
		Addr string
		// 是否启动ssl
		Secure bool
		// ca证书
		CaCert             string
		CaCertBase64       string
		ServerName         string
		Apikey             string
		CompressionEnabled bool
		AgentId            string
		Workspace          string
	}
	// registry 服务
	Service struct {
		config                        Config
		conn                          *grpc.ClientConn
		stop                          chan struct{}
		mutex                         sync.Mutex
		listeners                     []ReconnectListener
		lastGetControlConfigsResponse *pb2.GetControlConfigsResponse
		eventManager                  *eventManager
	}
	ReconnectListener interface {
		OnReconnect()
	}
)

func New(config Config) (*Service, error) {
	if !strings.Contains(config.Addr, ":") {
		config.Addr += ":7202"
	}
	rs := &Service{
		config: config,
		stop:   make(chan struct{}),
	}

	// 连接创建失败 直接返回错误
	// tcp上的失败不算
	err := rs.reconnect()
	if err != nil {
		return nil, err
	}

	rs.syncControlConfigsOnce()
	go rs.internalLoop()

	rs.eventManager = newEventManager(rs.reportEventBatchSync)
	rs.eventManager.start()

	return rs, nil
}

// RegisterAgent 注册agent
func (s *Service) RegisterAgent(ctx context.Context, agentInfo AgentInfo) (*pb2.RegisterAgentResponse, error) {
	resp, err := pb2.NewRegistryServiceForAgentClient(s.conn).RegisterAgent(ctx, &pb2.RegisterAgentRequest{
		Header:       s.createReqHeader(),
		AgentId:      agentInfo.AgentId,
		AgentVersion: agentInfo.AgentVersion,
		Ip:           agentInfo.Ip,
		Hostname:     agentInfo.Hostname,
		Os:           agentInfo.OS,
		Arch:         agentInfo.Arch,
		Labels:       agentInfo.Labels,
		App:          agentInfo.App,
		K8S:          agentInfo.K8s,
		Mode:         agentInfo.Mode,
		Workspace:    agentInfo.Workspace,
	}, s.getCallOptions()...)
	return resp, err
}

// SendHeartBeat 发送心跳
func (s *Service) SendHeartBeat(ctx context.Context, agentId string) (*pb2.SendAgentHeartbeatResponse, error) {
	// TODO 这个请求其实可以不压缩, 因为它太小了, 压缩效果不好反而浪费
	resp, err := pb2.NewRegistryServiceForAgentClient(s.conn).SendAgentHeartbeat(ctx, &pb2.SendAgentHeartbeatRequest{
		Header:  s.createReqHeader(),
		AgentId: agentId,
	}, s.getCallOptions()...)
	return resp, err
}

func (s *Service) getCallOptions() []grpc.CallOption {
	// TODO logo定了吗?
	// TODO 不想要它依赖appmeta
	md := metadata.Pairs(
		"x-cm-agent-id", agentmeta.GetAgentId(), //
		"x-cm-api-key", s.config.Apikey, //
	)
	return []grpc.CallOption{grpc.Header(&md)}
}

// GetControlConfigs 查控制参数
// TODO 我们要定义控制参数的模型, 转换后返回给用户, 而不是直接返回pb的结果给用户(虽然这样方便一点)
func (s *Service) GetControlConfigs(ctx context.Context, agentId string) (*pb2.GetControlConfigsResponse, error) {
	resp, err := pb2.NewRegistryServiceForAgentClient(s.conn).GetControlConfigs(ctx, &pb2.GetControlConfigsRequest{
		Header:  s.createReqHeader(),
		AgentId: agentId,
	}, s.getCallOptions()...)
	return resp, err
}

// GetCollectTasks 查询所有的采集配置
func (s *Service) GetCollectTasks(ctx context.Context, agentId string, buckets map[string]string) (*pb2.GetCollectTasksResponse, error) {
	resp, err := pb2.NewRegistryServiceForAgentClient(s.conn).GetCollectTasks(ctx, &pb2.GetCollectTasksRequest{
		Header:  s.createReqHeader(),
		AgentId: agentId,
		Buckets: buckets,
	}, s.getCallOptions()...)
	// TODO 模型转换
	return resp, err
}

// 上报运行状况统计
func (s *Service) ReportRunStats(ctx context.Context, agentId string, ts int64,
	agentRunStats map[string]uint64, collectConfigRunStats []*pb2.CollectConfigRunStats) (*pb2.RunStatsResponse, error) {
	resp, err := pb2.NewRegistryServiceForAgentClient(s.conn).SendRunStats(ctx, &pb2.RunStatsRequest{
		Header:                s.createReqHeader(),
		AgentId:               agentId,
		Timestamp:             ts,
		AgentRunStats:         agentRunStats,
		CollectConfigRunStats: collectConfigRunStats,
	}, s.getCallOptions()...)
	return resp, err
}

func (s *Service) ReportRunEvents(ctx context.Context, agentId string, events []*eventcollector.Event) (*pb2.RunEventsResponse, error) {
	// 解耦类型
	evs := make([]*pb2.RunEvent, 0, len(events))
	for _, v := range events {
		evs = append(evs, &pb2.RunEvent{Type: v.Type, Msg: v.Msg, Timestamp: v.Timestamp})
	}
	resp, err := pb2.NewRegistryServiceForAgentClient(s.conn).SendRunEvents(ctx, &pb2.RunEventsRequest{
		Header:  s.createReqHeader(),
		AgentId: agentId,
		Events:  evs,
	}, s.getCallOptions()...)
	return resp, err
}

// BiStream 双向流
func (s *Service) BiStream(ctx context.Context) (pb2.RegistryServiceForAgent_BiStreamsClient, error) {
	return pb2.NewRegistryServiceForAgentClient(s.conn).BiStreams(ctx, s.getCallOptions()...)
}

// Ping ping一下
func (s *Service) Ping(ctx context.Context) error {
	_, err := pb2.NewRegistryServiceForAgentClient(s.conn).Ping(ctx, &emptypb.Empty{})
	return err
}

// 创建通用请求头
func (s *Service) createReqHeader() *commonpb.CommonRequestHeader {
	return &commonpb.CommonRequestHeader{
		Apikey:    s.config.Apikey,
		Workspace: s.config.Workspace,
	}
}

// 这里如果返回错误, 则基本上是比较严重的错误, 比如证书不存在, 格式错误, 目标地址写错之类
func newConn(config Config) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	// 对 reg 由于要建立 双端流 因此需要尽力保持连接总是可用, 要及时keepalive
	// WithConnectParams
	// WithKeepaliveParams

	opts = append(opts, //
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  300 * time.Millisecond,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   3 * time.Second,
			},
			MinConnectTimeout: 20 * time.Second,
		}), //
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))

	if config.Secure {
		var cred credentials.TransportCredentials
		var err error
		if config.CaCert != "" {
			cred, err = credentials.NewClientTLSFromFile(config.CaCert, config.ServerName)
		} else {
			if config.CaCertBase64 != "" {
				cred, err = util.NewClientTLSFromBase64(config.CaCertBase64, config.ServerName)
			} else {
				cred, err = credentials.NewTLS(&tls.Config{ServerName: config.ServerName}), nil
			}
		}
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(cred))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, grpc.WithDefaultServiceConfig(`
{
  "methodConfig": [
    {
      "name": [
        {
          "service": "io.holoinsight.server.registry.grpc.agent.RegistryServiceForAgent"
        }
      ],
      "retryPolicy": {
        "maxAttempts": 3,
        "initialBackoff": "0.3s",
        "maxBackoff": "3s",
        "backoffMultiplier": 2,
        "retryableStatusCodes": [
          "UNAVAILABLE"
        ]
      }
    }
  ]
}`))

	// TODO 配置重试

	return grpc.DialContext(context.Background(), config.Addr, opts...)
}

// 定期重连
func (s *Service) internalLoop() {
	// 这个参数需要动态吗? 这样参数有点多
	ticker := time.NewTicker(reconnectInterval)
	defer ticker.Stop()

	syncControlConfigTimer := time.NewTicker(syncControlConfigsInterval)
	defer syncControlConfigTimer.Stop()

	for {
		select {
		case <-ticker.C:
			s.reconnect()
		case <-syncControlConfigTimer.C:
			s.syncControlConfigsOnce()
			syncControlConfigTimer.Reset(syncControlConfigsInterval)
		case <-s.stop:
			return
		}
	}
}

// 定期重连
func (s *Service) reconnect() error {
	conn, err := newConn(s.config)
	if err != nil {
		// 打印日志
		logger.Errorf("create registry grpc conn error %+v", err)
		return err
	}

	{
		// 快速尝试ping一下, 失败了无所谓
		ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
		defer cancel()
		err = ping(ctx, conn)
		// 这个链接不健康
		if err != nil {
			logger.Errorf("new conn ping error %+v", err)
		}

	}

	old := s.conn
	s.conn = conn

	var listeners []ReconnectListener
	s.mutex.Lock()
	listeners = s.listeners
	s.mutex.Unlock()

	for _, listener := range listeners {
		listener.OnReconnect()
	}

	// TODO 这里有个问题: 需要想办法尽量平滑的切换 bistreams

	if old != nil {
		// 延迟关闭, 足够安全了
		s.delayClose(old)
	}

	return nil
}

func ping(ctx context.Context, conn *grpc.ClientConn) error {
	// 先ping一把
	client := pb2.NewRegistryServiceForAgentClient(conn)
	_, err := client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (s *Service) isStopped() bool {
	select {
	case <-s.stop:
		return true
	default:
		return false
	}
}

func (s *Service) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.isStopped() {
		return
	}

	close(s.stop)
	// delay close the underlying conn
	time.AfterFunc(delayCloseDuration, func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		s.delayClose(s.conn)
		s.conn = nil
	})
	s.eventManager.stop()
}

func (s *Service) ListenReconnect(listener ReconnectListener) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	newListeners := s.listeners
	newListeners = append(newListeners, listener)
	s.listeners = newListeners
}

func (s *Service) RemoveReconnect(listener ReconnectListener) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i := range s.listeners {
		if s.listeners[i] == listener {
			size := len(s.listeners)
			newListeners := make([]ReconnectListener, size-1)
			copy(newListeners[:i], s.listeners[:i])
			copy(newListeners[i:size-1], s.listeners[i+1:size])
			s.listeners = newListeners
			return
		}
	}
}

func (s *Service) syncControlConfigsOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := s.GetControlConfigs(ctx, s.config.AgentId)
	if err != nil {
		logger.Errorz("[registry] GetControlConfigs error", zap.Error(err))
		return
	}
	last := s.lastGetControlConfigsResponse
	if last != nil && resp != nil {
		lastBytes, _ := proto.Marshal(last)
		respBytes, _ := proto.Marshal(resp)
		if !bytes.Equal(lastBytes, respBytes) {
			logger.Configz("[registry] ControlConfigs changed", zap.Any("old", last), zap.Any("new", resp))
		}
	}
	s.lastGetControlConfigsResponse = resp
}

func (s *Service) GetLastControlConfigs() *pb2.GetControlConfigsResponse {
	return s.lastGetControlConfigsResponse
}

func (s *Service) MetaFullSync(request *pb2.MetaSync_FullSyncRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opts := s.getCallOptions()
	request.Header = s.createReqHeader()
	_, err := pb2.NewRegistryServiceForAgentClient(s.conn).MetaFullSync(ctx, request, opts...)
	return err
}

func (s *Service) MetaDeltaSync(request *pb2.MetaSync_DeltaSyncRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opts := s.getCallOptions()
	request.Header = s.createReqHeader()
	_, err := pb2.NewRegistryServiceForAgentClient(s.conn).MetaDeltaSync(ctx, request, opts...)
	return err
}

func (s *Service) ReportEventAsync(events ...*pb2.ReportEventRequest_Event) {
	s.eventManager.add(events)
}

func (s *Service) reportEventBatchSync(events []*pb2.ReportEventRequest_Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opts := s.getCallOptions()
	request := &pb2.ReportEventRequest{Events: events}
	request.Header = s.createReqHeader()
	_, err := pb2.NewRegistryServiceForAgentClient(s.conn).ReportEvents(ctx, request, opts...)
	return err
}

func (s *Service) delayClose(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}
	time.AfterFunc(delayCloseDuration, func() {
		conn.Close()
	})
}
