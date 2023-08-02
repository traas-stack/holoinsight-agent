package grpcutil

import (
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
	"time"
)

type (
	CreateConnFunc func() (*grpc.ClientConn, error)
	ConnPool       struct {
		create            CreateConnFunc
		size              int
		conns             []*grpc.ClientConn
		index             int64
		mutex             sync.Mutex
		stopCh            chan struct{}
		reconnectInterval time.Duration
		closeDelay        time.Duration
	}
)

func NewConnPool(create CreateConnFunc, size int, reconnectInterval time.Duration, closeDelay time.Duration) *ConnPool {
	return &ConnPool{
		create:            create,
		size:              size,
		stopCh:            make(chan struct{}),
		reconnectInterval: reconnectInterval,
		closeDelay:        closeDelay,
	}
}

func (p *ConnPool) Get() *grpc.ClientConn {
	i := atomic.AddInt64(&p.index, 1)
	index := int(i % int64(p.size))
	return p.conns[index]
}

func (p *ConnPool) Start() {
	p.rebuild()
	go p.loop()
}

func (p *ConnPool) isStopped() bool {
	select {
	case <-p.stopCh:
		return true
	default:
		return false
	}
}

func (p *ConnPool) stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isStopped() {
		return
	}

	close(p.stopCh)
	p.delayClose(p.conns)
}

func (p *ConnPool) rebuild() {
	old := p.conns
	conns := make([]*grpc.ClientConn, p.size)
	for i := 0; i < p.size; i++ {
		conn, err := p.create()
		if err != nil {
			panic(err)
		}
		conn.Connect()
		conns[i] = conn
	}
	p.conns = conns

	p.delayClose(old)
}

func (p *ConnPool) delayClose(conns []*grpc.ClientConn) {
	if len(conns) > 0 {
		time.AfterFunc(p.closeDelay, func() {
			for _, conn := range conns {
				conn.Close()
			}
		})
	}
}

func (p *ConnPool) loop() {
	timer := time.NewTimer(p.reconnectInterval)
	defer timer.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-timer.C:
			func() {
				p.mutex.Lock()
				defer p.mutex.Unlock()
				if p.isStopped() {
					return
				}
				p.rebuild()
				timer.Reset(p.reconnectInterval)
			}()
		}
	}
}
