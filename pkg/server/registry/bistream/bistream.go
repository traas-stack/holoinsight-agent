package bistream

import (
	"context"
	"errors"
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	commonpb "github.com/TRaaSStack/holoinsight-agent/pkg/server/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"io"
	"sync/atomic"
	"time"
)

// TODO 太复杂 简化一下

const (
	RpcTypeOneway          = 0
	RpcTypeReq             = 1
	RpcTypeResp            = 2
	RpcTypeClientHandShake = 3
	RpcTypeServerHandShake = 4
	BizTypeNotFound        = -1
	BizTypeError           = -2
)

type (
	Manager struct {
		rs       *registry.Service
		hr       *registry.HandlerRegistry
		stop     chan struct{}
		listener *reconnectListener
		eventCh  chan interface{}
		si       *streamInstance
	}
	streamInstance struct {
		version int64
		m       *Manager
		c       pb.RegistryServiceForAgent_BiStreamsClient
		stopCh  chan struct{}
	}
	reconnectListener struct {
		m *Manager
	}
)

func NewManager(rs *registry.Service, hr *registry.HandlerRegistry) *Manager {
	m := &Manager{
		rs:       rs,
		hr:       hr,
		stop:     make(chan struct{}),
		listener: &reconnectListener{},
		eventCh:  make(chan interface{}, 16),
	}
	m.listener.m = m
	return m
}

func (m *Manager) Start() {
	logger.Infoz("[bistreammanager] start")
	go m.loop()
	m.rs.ListenReconnect(m.listener)
}

func (m *Manager) Stop() {
	logger.Infoz("[bistreammanager] stop")
	m.rs.RemoveReconnect(m.listener)
	close(m.stop)
	logger.Infoz("[bistreammanager] stopped")
}

func (m *Manager) onReconnect() {
	// reg将要重新建连了
	select {
	case <-m.stop:
		return
	default:
		m.eventCh <- reconnectEvent(0)
	}

}

type (
	reconnectEvent int
	runEvent       struct {
		f      func()
		result chan struct{}
	}
)

func (m *Manager) loop() {
	m.si, _ = m.startStreamInstance(false)

	for {
		select {
		case e := <-m.eventCh:
			switch x := e.(type) {
			case runEvent:
				func() {
					defer func() {
						x.result <- struct{}{}
					}()
					x.f()
				}()
			case reconnectEvent:
				si, err := m.startStreamInstance(true)
				if err != nil {
					logger.Errorz("[bistream] fail to open stream", zap.Error(err))
					continue
				}
				old := m.si
				m.si = si
				if old != nil {
					time.AfterFunc(time.Second, old.stop)
				}
			}
		case <-m.stop:
			return
		}
	}
}

func (m *Manager) startStreamInstance(handshake bool) (*streamInstance, error) {
	si := &streamInstance{
		version: util.CurrentMS(),
		m:       m,
		stopCh:  make(chan struct{}),
	}
	// 立即握手
	if handshake {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := si.handshake(ctx)
		cancel()
		if err != nil {
			logger.Errorz("[bistream] handshake error", zap.Error(err))
			return nil, err
		}
	}
	go si.start()
	return si, nil
}

func (si *streamInstance) stop() {
	close(si.stopCh)
}

func (si *streamInstance) start() {
	for {
		select {
		case <-si.stopCh:
			return
		default:
		}

		if si.c == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := si.handshake(ctx)
			cancel()
			if err != nil {
				logger.Errorz("[bistream] handshake error", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}
		}
		err := si.serveOnce()
		if err != nil {
			logger.Errorz("[bistream] stream closed", zap.Error(err))
		} else {
			logger.Infoz("[bistream] stream closed")
		}
		si.c = nil
	}
}

func (si *streamInstance) serveOnce() error {
	logger.Infoz("[bistream] started")

	c := si.c
	hr := si.m.hr

	sendCh := make(chan *commonpb.GenericRpcCommand, 256)

	// 初始值为1
	pendings := int32(1)
	wipCloseSendCh := func() {
		if atomic.AddInt32(&pendings, -1) == 0 {
			close(sendCh)
		}
	}
	send := func(resp *commonpb.GenericRpcCommand) bool {
		select {
		// 99% case
		case sendCh <- resp:
			return true
		default:
			timer := time.NewTimer(3 * time.Second)
			defer timer.Stop()
			select {
			case sendCh <- resp:
				return true
			case <-timer.C:
				return false
			}
		}
	}
	drainSendCh := func() {
		for range sendCh {
		}
	}
	// consume send ch
	go func() {
		for {
			select {
			case <-si.stopCh:
				logger.Infoz("[bistream] stream closed, ignore sending")
				if err := c.CloseSend(); err != nil {
					logger.Errorz("[bistream] close stream", zap.Error(err))
				}
				drainSendCh()
				return
			case resp := <-sendCh:
				// logger.Infoz("c.send", zap.Stringer("resp", resp))
				if err := c.Send(resp); err != nil {
					logger.Infoz("[bistream] c.send error, close stream", zap.Error(err))
					if err := c.CloseSend(); err != nil {
						logger.Errorz("[bistream] close stream", zap.Error(err))
					}
					drainSendCh()
					return
				}
			}
		}
	}()

	// 关注这个for循环, 它是我们所有的请求的来源, 是sendCh的根本来源
	// 因此只要这个地方停了, 那么sendCh就可以安全地关闭了
	for {
		select {
		case <-si.stopCh:
			wipCloseSendCh()
			return nil
		default:
		}

		req, err := c.Recv()
		if err != nil {
			logger.Infoz("[bistream] recv error", zap.Error(err))
			wipCloseSendCh()
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch req.RpcType {
		case RpcTypeOneway:
			h := hr.GetOneway(req.BizType)
			if h != nil {
				go h(req.BizType, req.Data)
			}
			// else just ignore
		case RpcTypeReq:
			h := hr.GetRpc(req.BizType)
			if h != nil {
				atomic.AddInt32(&pendings, 1)
				go func() {
					defer wipCloseSendCh()

					respBizType, resp, err := h(req.BizType, req.Data)
					var b []byte
					if err == nil {
						switch x := resp.(type) {
						case []byte:
							b = x
						case proto.Message:
							b, err = proto.Marshal(x)
						default:
							err = errors.New("unsupported response type")
						}
					}
					if err != nil {
						send(&commonpb.GenericRpcCommand{
							RpcType: RpcTypeResp,
							ReqId:   req.ReqId,
							BizType: BizTypeError,
							Data:    []byte(fmt.Sprintf("biz error: %+v", err)),
						})
					} else {
						send(&commonpb.GenericRpcCommand{
							RpcType: RpcTypeResp,
							ReqId:   req.ReqId,
							BizType: respBizType,
							Data:    b,
						})
					}
				}()
			} else {
				send(&commonpb.GenericRpcCommand{
					RpcType: RpcTypeResp,
					ReqId:   req.ReqId,
					BizType: BizTypeNotFound,
					Data:    []byte(fmt.Sprintf("unsupported type: %d", req.BizType)),
				})
			}
		default:
		}
	}
}

func (si *streamInstance) handshake(ctx context.Context) error {
	rs := si.m.rs
	c, err := rs.BiStream(context.Background())
	if err != nil {
		// backoff
		logger.Errorz("[bistream] fail to open stream", zap.Error(err))
		return err
	}

	h := si.m.hr.GetHandshakeHandler()
	if h == nil {
		c.CloseSend()
		return errors.New("[bistream] miss HandshakeHandler")
	}

	err = h(ctx, c, si.version)
	if err == nil {
		si.c = c
	}
	return err
}

func (r *reconnectListener) OnReconnect() {
	r.m.onReconnect()
}
