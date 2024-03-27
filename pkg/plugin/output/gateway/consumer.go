/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package gateway

import (
	"context"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"go.uber.org/zap"
	"strings"
	"time"
)

type (
	batchConsumerV1 struct {
		gw *gateway.Service
	}
	batchConsumerV4 struct {
		gw        *gateway.Service
		semaphore chan struct{}
	}
	result struct {
		Resp *pb.WriteMetricsResponse
		Err  error
	}
	// taskV1 is the write requests
	taskV1 struct {
		batch []*model.Metric
		// resultCh is the result chan
		// If it is nil, it means no need to response
		resultCh chan *result
	}
	// taskV4 is the write requests
	taskV4 struct {
		batch []*pb.WriteMetricsRequestV4_TaskResult
		// resultCh is the result chan
		// If it is nil, it means no need to response
		resultCh chan *result
	}
)

var (
	gatewayDiscardStat = stat.DefaultManager1S.Counter("gateway.discard")
	gatewaySendStat    = stat.DefaultManager1S.Counter("gateway.send")
)

func (b *batchConsumerV1) Consume(a []interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var points []*model.Metric
	for _, i := range a {
		switch x := i.(type) {
		case *taskV1:
			points = append(points, x.batch...)
		}
	}
	begin := time.Now()
	resp, err := b.gw.WriteMetricsV1Extension2(ctx, nil, points)
	cost := time.Now().Sub(begin)
	if err == nil && resp.Header.Code != 0 {
		err = fmt.Errorf("server error %+v", resp.Header)
	}

	if err != nil {
		gatewaySendStat.Add([]string{"v1", "N"}, []int64{1, int64(len(a)), int64(len(points)), cost.Milliseconds()})
	} else {
		gatewaySendStat.Add([]string{"v1", "Y"}, []int64{1, int64(len(a)), int64(len(points)), cost.Milliseconds()})
	}

	taskResult := &result{
		Resp: resp,
		Err:  err,
	}
	for _, i := range a {
		switch x := i.(type) {
		case *taskV1:
			if x.resultCh != nil {
				x.resultCh <- taskResult
				close(x.resultCh)
			}
		}
	}
}

func (b *batchConsumerV4) Consume(a []interface{}) {
	// copy one
	a = append([]interface{}(nil), a...)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	var taskResults []*pb.WriteMetricsRequestV4_TaskResult
	points := 0
	for _, i := range a {
		switch x := i.(type) {
		case *taskV4:
			taskResults = append(taskResults, x.batch...)
			for _, tr := range x.batch {
				points += len(tr.Table.Rows)
			}
		}
	}
	begin := time.Now()

	b.semaphore <- struct{}{}
	go func() {
		defer func() {
			cancel()
			<-b.semaphore
		}()
		var err error
		var resp *pb.WriteMetricsResponse

		for i := 0; i < 3; i++ {
			resp, err = b.gw.WriteMetrics(ctx, taskResults)
			if err != nil && strings.Contains(err.Error(), "connection refused") {
				time.Sleep(300 * time.Millisecond)
				continue
			}
			break
		}

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

		taskResult := &result{
			Resp: resp,
			Err:  err,
		}
		for _, i := range a {
			switch x := i.(type) {
			case *taskV4:
				if x.resultCh != nil {
					x.resultCh <- taskResult
					close(x.resultCh)
				}
			}
		}
	}()
}
