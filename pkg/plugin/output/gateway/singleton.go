/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package gateway

import (
	"context"
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
)

var gatewayDiscardStat = stat.DefaultManager.Counter("gateway.discard")

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
	processor := batch.NewBatchProcessor(1024, &batchConsumer{
		gw: gw,
	}, batch.WithMaxWaitStrategy(time.Second),
		batch.WithItemsWeightStrategy(func(i interface{}) int {
			if tr, ok := i.(*pb.WriteMetricsRequestV4_TaskResult); ok {
				return len(tr.Table.Rows)
			} else {
				return 1
			}
		}, 1024))

	processor.Run()

	return []interface{}{processor, gw}, nil
}, func(i interface{}) {
	i.(batch.Processor).Shutdown()
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
	results := make([]*pb.WriteMetricsRequestV4_TaskResult, len(a))
	for i, tr := range a {
		results[i] = tr.(*pb.WriteMetricsRequestV4_TaskResult)
	}
	_, err := b.gw.WriteMetrics(ctx, results)
	if err != nil {
		logger.Errorz("[gateway] write error", zap.Error(err))
		// 统计丢数据数量
		gatewayDiscardStat.Add(nil, []int64{ //
			int64(len(a)), //
		})
	}
}
