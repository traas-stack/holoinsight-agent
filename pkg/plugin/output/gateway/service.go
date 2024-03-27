/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package gateway

import (
	"context"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/server/fordev"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util/batch"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"sync"
	"time"
)

const (
	defaultWriteBatchPointSize = 4096
	defaultWriteQueueSize      = 4096
	defaultSemaphore           = 64
	defaultWriteBatchWait      = 500 * time.Millisecond
)

type (
	WriteV1Request struct {
		Batch     []*model.Metric
		Extension map[string]string
		NoMerge   bool
	}
	WriteV4Request struct {
		Batch []*pb.WriteMetricsRequestV4_TaskResult
	}
	WriteService interface {
		WriteV1(ctx context.Context, req *WriteV1Request) error
		WriteV4(ctx context.Context, req *WriteV4Request) error
	}
	writeServiceImpl struct {
		ensureGatewayInitedOnce sync.Once
		gateway                 *gateway.Service
		bpV1                    batch.Processor
		bpV4                    batch.Processor
	}
)

var (
	writeService             WriteService = &writeServiceImpl{}
	errGatewayWriteQueueFull              = errors.New("gateway write queue full")
)

func GetWriteService() WriteService {
	return writeService
}

func (w *writeServiceImpl) ensureGatewayInited() {
	w.ensureGatewayInitedOnce.Do(w.ensureGatewayInited0)
}

func (w *writeServiceImpl) ensureGatewayInited0() {
	gateway, err := fordev.NewDevGateway()
	if err != nil {
		logger.Errorz("[gateway] init error")
		return
	}

	bpV1 := batch.NewBatchProcessor(defaultWriteQueueSize, &batchConsumerV1{gw: gateway},
		batch.WithMaxWaitStrategy(defaultWriteBatchWait),
		batch.WithItemsWeightStrategy(func(i interface{}) int {
			switch x := i.(type) {
			case *taskV1:
				return len(x.batch)
			default:
				return 1
			}
		}, defaultWriteBatchPointSize))
	bpV1.Run()

	bpV4 := batch.NewBatchProcessor(defaultWriteQueueSize, &batchConsumerV4{gw: gateway, semaphore: make(chan struct{}, defaultSemaphore)},
		batch.WithMaxWaitStrategy(defaultWriteBatchWait),
		batch.WithItemsWeightStrategy(func(i interface{}) int {
			switch x := i.(type) {
			case *taskV4:
				sum := 0
				for _, tr := range x.batch {
					sum += len(tr.Table.Rows)
				}
				return sum
			default:
				return 1
			}
		}, defaultWriteBatchPointSize))
	bpV4.Run()

	w.gateway = gateway
	w.bpV1 = bpV1
	w.bpV4 = bpV4

	stat.DefaultManager1S.Gauge("bpv1.pending", func() []stat.GaugeSubItem {
		return []stat.GaugeSubItem{
			{
				Values: []int64{int64(bpV1.Num())},
			},
		}
	})
	stat.DefaultManager1S.Gauge("bpv4.pending", func() []stat.GaugeSubItem {
		return []stat.GaugeSubItem{
			{
				Values: []int64{int64(bpV4.Num())},
			},
		}
	})
}

func (w *writeServiceImpl) WriteV1(ctx context.Context, req *WriteV1Request) error {
	w.ensureGatewayInited()

	if len(req.Extension) > 0 || req.NoMerge {
		resp, err := w.gateway.WriteMetricsV1Extension2(ctx, req.Extension, req.Batch)
		if err == nil && resp.Header.Code != 0 {
			err = fmt.Errorf("server error: %+v", resp.Header)
		}
		return err
	}

	task := &taskV1{
		batch:    req.Batch,
		resultCh: make(chan *result, 1),
	}
	if !w.bpV1.TryPut(task) {
		return errGatewayWriteQueueFull
	}

	select {
	case result := <-task.resultCh:
		return result.Err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *writeServiceImpl) WriteV4(ctx context.Context, req *WriteV4Request) error {
	w.ensureGatewayInited()

	task := &taskV4{
		batch:    req.Batch,
		resultCh: make(chan *result, 1),
	}
	if !w.bpV4.TryPut(task) {
		return errGatewayWriteQueueFull

	}

	select {
	case result := <-task.resultCh:
		return result.Err
	case <-ctx.Done():
		return ctx.Err()
	}
}
