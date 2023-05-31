/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package gateway

import (
	"context"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/fordev"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util/batch"
	"github.com/traas-stack/holoinsight-agent/pkg/util/singleton"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"go.uber.org/zap"
	"time"
)

type (
	batchConsumer struct {
		gw *gateway.Service
	}
	Result struct {
		Resp *pb.WriteMetricsResponse
		Err  error
	}
	// TaskV4 is the write requests
	TaskV4 struct {
		Batch []*pb.WriteMetricsRequestV4_TaskResult
		// ResultCh is the result chan
		// If it is nil, it means no need to response
		ResultCh chan *Result
	}
)

var gatewayDiscardStat = stat.DefaultManager.Counter("gateway.discard")
var gatewaySendStat = stat.DefaultManager.Counter("gateway.send")

var GatewaySingletonHolder = singleton.NewHolder(func() (interface{}, error) {
	return fordev.NewDevGateway()
}, func(i interface{}) {
	i.(*gateway.Service).Stop()
})

var gatewayProcessorSingletonHolder = singleton.NewHolder(func() (interface{}, error) {
	gwI, err := GatewaySingletonHolder.Acquire()
	if err != nil {
		return nil, err
	}
	gw := gwI.(*gateway.Service)
	processor := batch.NewBatchProcessor(65536, &batchConsumer{
		gw: gw,
	}, batch.WithMaxWaitStrategy(time.Second),
		batch.WithItemsWeightStrategy(func(i interface{}) int {
			switch x := i.(type) {
			case *TaskV4:
				sum := 0
				for _, tr := range x.Batch {
					sum += len(tr.Table.Rows)
				}
				return sum
			default:
				return 1
			}
		}, 3072))

	processor.Run()

	return []interface{}{processor, gw}, nil
}, func(i interface{}) {
	i.([]interface{})[1].(batch.Processor).Shutdown()
	GatewaySingletonHolder.Release()
})

func Acquire() (*gateway.Service, error) {
	i, err := GatewaySingletonHolder.Acquire()
	if err != nil {
		return nil, err
	}
	return i.(*gateway.Service), nil
}

func (b *batchConsumer) Consume(a []interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var taskResults []*pb.WriteMetricsRequestV4_TaskResult
	points := 0
	for _, i := range a {
		switch x := i.(type) {
		case *TaskV4:
			taskResults = append(taskResults, x.Batch...)
			for _, tr := range x.Batch {
				points += len(tr.Table.Rows)
			}
		}
	}
	begin := time.Now()
	resp, err := b.gw.WriteMetrics(ctx, taskResults)
	cost := time.Now().Sub(begin)
	if err == nil && resp.Header.Code != 0 {
		err = fmt.Errorf("server error %+v", resp.Header)
	}

	if err != nil {
		logger.Errorz("[gateway] write error", zap.Error(err))
		// 统计丢数据数量
		gatewayDiscardStat.Add(nil, []int64{ //
			int64(len(a)), //
		})
		gatewaySendStat.Add([]string{"v4", "N"}, []int64{1, int64(len(a)), int64(points), cost.Milliseconds()})
	} else {
		gatewaySendStat.Add([]string{"v4", "Y"}, []int64{1, int64(len(a)), int64(points), cost.Milliseconds()})
	}

	taskResult := &Result{
		Resp: resp,
		Err:  err,
	}
	for _, i := range a {
		switch x := i.(type) {
		case *TaskV4:
			if x.ResultCh != nil {
				x.ResultCh <- taskResult
				close(x.ResultCh)
			}
		}
	}
}
