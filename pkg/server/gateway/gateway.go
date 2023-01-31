package gateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/appconfig"
	"github.com/TRaaSStack/holoinsight-agent/pkg/core"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	pb2 "github.com/TRaaSStack/holoinsight-agent/pkg/server/gateway/pb"
	commonpb "github.com/TRaaSStack/holoinsight-agent/pkg/server/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

const (
	pingTimeout        = 3 * time.Second
	reconnectInterval  = 10 * time.Minute
	delayCloseTimeout  = 1 * time.Minute
	compressThreshold  = 10
	defaultCompression = "gzip"
)

type (
	Config struct {
		Addr         string
		Secure       bool
		CaCert       string
		CaCertBase64 string
		ServerName   string
		Apikey       string
	}
	Service struct {
		config Config
		conn   *grpc.ClientConn
		stop   chan struct{}
	}
)

func New(config Config) (*Service, error) {
	conn, err := newConn(config)
	if err != nil {
		return nil, err
	}
	s := &Service{
		config: config,
		conn:   conn,
		stop:   make(chan struct{}),
	}
	return s, nil
}

func (s *Service) Start() {
	// 立即触发一个grpc请求, 如果失败了只是记录一下日志不做其他处理
	// 为的是不让conn为nil 利用grpc的自动重连功能 可能可以恢复正常
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	ping(ctx, s.conn)
	go s.reconnectLoop()
}

func ping(ctx context.Context, conn *grpc.ClientConn) {
	c := pb2.NewGatewayServiceClient(conn)
	_, err := c.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		logger.Errorf("ping gateway error %+v", err)
	}
}

func (s *Service) reconnectLoop() {
	timer := time.NewTimer(reconnectInterval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			s.reconnect()
			timer.Reset(reconnectInterval)
		case <-s.stop:
			conn := s.conn
			if conn != nil {
				conn.Close()
			}
			break
		}
	}
}

func (s *Service) reconnect() {
	conn, err := newConn(s.config)
	if err != nil {
		logger.Errorf("reconnect error %+v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	ping(ctx, conn)

	old := s.conn
	s.conn = conn

	delayClose(old)
}

func delayClose(conn *grpc.ClientConn) {
	if conn != nil {
		time.AfterFunc(delayCloseTimeout, func() {
			conn.Close()
		})
	}
}

func (s *Service) Stop() {
	conn := s.conn
	s.conn = nil
	delayClose(conn)
	close(s.stop)
}

func (s *Service) client() pb2.GatewayServiceClient {
	return pb2.NewGatewayServiceClient(s.conn)
}

func (s *Service) WriteMetricsV1Extension(ctx context.Context, extension map[string]string, points []*pb2.Point) (*pb2.WriteMetricsResponse, error) {
	// TODO 原地修改是不安全的
	if appconfig.StdAgentConfig.Mode == core.AgentModeDaemonset {
		for _, point := range points {
			point.Tags["workspace"] = appconfig.StdAgentConfig.Workspace
		}
	}

	req := &pb2.WriteMetricsRequestV1{
		Header: &commonpb.CommonRequestHeader{
			Apikey: s.config.Apikey,
			Header: extension,
		},
		Point: points,
	}
	return s.client().WriteMetricsV1(ctx, req)
}

func (s *Service) WriteMetricsV1Extension2(ctx context.Context, extension map[string]string, metrics []*model.Metric) (*pb2.WriteMetricsResponse, error) {
	// TODO 原地修改是不安全的
	if appconfig.StdAgentConfig.Mode == core.AgentModeDaemonset {
		for _, point := range metrics {
			point.Tags["workspace"] = appconfig.StdAgentConfig.Workspace
		}
	}

	points := make([]*pb2.Point, len(metrics))
	for i, metric := range metrics {
		points[i] = &pb2.Point{
			MetricName: metric.Name,
			Timestamp:  metric.Timestamp,
			Tags:       metric.Tags,
			NumberValues: map[string]float64{
				"value": metric.Value,
			},
			StringValues: nil,
		}
	}
	req := &pb2.WriteMetricsRequestV1{
		Header: &commonpb.CommonRequestHeader{
			Apikey: s.config.Apikey,
			Header: extension,
		},
		Point: points,
	}

	var callOpts []grpc.CallOption
	if len(points) > compressThreshold {
		callOpts = append(callOpts, grpc.UseCompressor(defaultCompression))
	}
	return s.client().WriteMetricsV1(ctx, req, callOpts...)
}

func (s *Service) WriteMetrics(ctx context.Context, results []*pb2.WriteMetricsRequestV4_TaskResult) (*pb2.WriteMetricsResponse, error) {
	// TODO 修改tags
	req := &pb2.WriteMetricsRequestV4{
		Header: &commonpb.CommonRequestHeader{
			Apikey: s.config.Apikey,
		},
		Results: results,
	}

	var callOpts []grpc.CallOption
	callOpts = append(callOpts, grpc.UseCompressor(defaultCompression))
	return s.client().WriteMetricsV4(ctx, req, callOpts...)
}

func (s *Service) Ping(ctx context.Context) error {
	_, err := s.client().Ping(ctx, &emptypb.Empty{})
	return err
}

func newConn(config Config) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithConnectParams(grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  300 * time.Millisecond,
			Multiplier: 1.6,
			Jitter:     0.2,
			MaxDelay:   10 * time.Second,
		},
		// 默认就是20 防止不小心覆盖了
		MinConnectTimeout: 20 * time.Second,
	}), grpc.WithKeepaliveParams(keepalive.ClientParameters{
		// https://www.jianshu.com/p/97d465428a89
		// grpc 其实会进行错误重连的 所以其实keepalive不是很有必要, TODO 验证一下这句话
		// 1min没有包就ping一下
		Time:                1 * time.Minute,
		Timeout:             20 * time.Second,
		PermitWithoutStream: false,
	}), grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(64*1024*1024),
		grpc.UseCompressor("gzip")), grpc.WithDefaultServiceConfig(`
{
  "methodConfig": [
    {
      "name": [
        {
          "service": "io.holoinsight.server.gateway.grpc.GatewayService"
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

	if config.Secure {
		var cred credentials.TransportCredentials = nil
		var err error
		if config.CaCert != "" {
			cred, err = credentials.NewClientTLSFromFile(config.CaCert, config.ServerName)
		} else {
			var rootCas *x509.CertPool = nil
			if config.CaCertBase64 != "" {
				b, err := base64.StdEncoding.DecodeString(config.CaCertBase64)
				if err != nil {
					return nil, err
				}
				rootCas = x509.NewCertPool()
				if !rootCas.AppendCertsFromPEM(b) {
					return nil, fmt.Errorf("credentials: failed to append certificates")
				}
			}
			cred, err = credentials.NewTLS(&tls.Config{ServerName: config.ServerName, RootCAs: rootCas}), nil
		}
		if err != nil {
			return nil, err
		}

		opts = append(opts, grpc.WithTransportCredentials(cred))

	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	return grpc.DialContext(context.Background(), config.Addr, opts...)
}
