/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package gateway

import (
	"context"
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
	"github.com/traas-stack/holoinsight-agent/pkg/util/batch"
)

const (
	defaultTimeout = 5 * time.Second
)

type (
	gatewayOutput struct {
		processor batch.Processor
		service   *gateway.Service
	}
)

func (c *gatewayOutput) WriteMetrics(metrics []*model.Metric, oe output.Extension) {
	gI, err := GatewaySingletonHolder.Acquire()
	if err != nil {
		return
	}
	defer GatewaySingletonHolder.Release()

	g := gI.(*gateway.Service)
	extension := map[string]string{
		"tenant": oe.Tenant,
	}
	resp, err := g.WriteMetricsV1Extension2(context.Background(), extension, metrics)
	if err != nil || resp.Header.Code != 0 {
		logger.Errorf("WriteMetricsV1Extension2 %+v %+v", resp, err)
	}
}

func newGatewayOutput(config output.Config) (output.Output, error) {
	iiI, err := gatewayProcessorSingletonHolder.Acquire()
	if err != nil {
		return nil, err
	}
	ii := iiI.([]interface{})
	processor := ii[0].(batch.Processor)
	return &gatewayOutput{
		processor: processor,
		service:   ii[1].(*gateway.Service),
	}, nil
}

func (c *gatewayOutput) Start() {

}

func (c *gatewayOutput) Stop() {
	gatewayProcessorSingletonHolder.Release()
}

func (c *gatewayOutput) WriteBatchAsync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	go c.writeBatchAsync0(configKey, targetKey, metricName, array)
	return nil
}

var outputStat = stat.DefaultManager1S.Counter("output.gateway")

func (c *gatewayOutput) WriteBatchSync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	converted := convertToTaskResult2(configKey, targetKey, metricName, array)
	task := &TaskV4{
		Batch:    converted,
		ResultCh: make(chan *Result, 1),
	}

	if !c.processor.TryPut(task) {
		return errors.New("write queue full")
	}
	return (<-task.ResultCh).Err
}

func (c *gatewayOutput) writeBatchAsync0(configKey, targetKey, metricName string, array []*model.DetailData) error {
	converted := convertToTaskResult2(configKey, targetKey, metricName, array)
	outputStat.Add([]string{
		configKey,
	}, []int64{
		int64(len(converted)),
	})
	task := &TaskV4{
		Batch:    converted,
		ResultCh: make(chan *Result, 1),
	}
	if !c.processor.TryPut(task) {
		return errors.New("write queue full")
	}
	return nil
}

var singleValueKeys = []string{"value"}

func getOrCreate(configKey, targetKey, metricName string, taskResultByValueName map[string]*pb.WriteMetricsRequestV4_TaskResult, dd *model.DetailData, valueName string) *pb.WriteMetricsRequestV4_TaskResult {
	if r, ok := taskResultByValueName[valueName]; ok {
		return r
	}

	var mergedMetricName string
	if valueName == "value" {
		mergedMetricName = metricName
	} else if strings.Contains(metricName, "%s") {
		mergedMetricName = strings.Replace(metricName, "%s", valueName, 1)
	} else if strings.HasSuffix(metricName, ".") || strings.HasSuffix(metricName, "_") {
		mergedMetricName = metricName + valueName
	} else {
		mergedMetricName = metricName + "_" + valueName
	}

	// Here we have to sort the tagKeys to make it order stable
	// tagKeys and tagValues must match.
	tagKeys := make([]string, 0, len(dd.Tags))
	for k := range dd.Tags {
		tagKeys = append(tagKeys, k)
	}
	sort.Strings(tagKeys)

	r := &pb.WriteMetricsRequestV4_TaskResult{
		Key:           configKey + "/" + targetKey,
		RefCollectKey: configKey,
		RefTargetKey:  targetKey,
		Table: &pb.WriteMetricsRequestV4_Table{
			Header: &pb.WriteMetricsRequestV4_Header{
				MetricName: mergedMetricName,
				TagKeys:    tagKeys,
				ValueKeys:  singleValueKeys,
			},
		},
	}
	taskResultByValueName[valueName] = r
	return r
}

func convertToTaskResult2(configKey, targetKey, metricName string, array []*model.DetailData) []*pb.WriteMetricsRequestV4_TaskResult {
	// datum in array must have same tag keys

	taskResultByValueName := make(map[string]*pb.WriteMetricsRequestV4_TaskResult)

	for _, i := range array {
		tagValues := make([]string, 0, len(i.Tags))

		for valueName, v := range i.Values {
			taskResult := getOrCreate(configKey, targetKey, metricName, taskResultByValueName, i, valueName)
			if len(tagValues) == 0 {
				for _, tagKey := range taskResult.Table.Header.TagKeys {
					tagValues = append(tagValues, i.Tags[tagKey])
				}
			}

			taskResult.Table.Rows = append(taskResult.Table.Rows, &pb.WriteMetricsRequestV4_Row{
				Timestamp: i.Timestamp,
				TagValues: tagValues,
				ValueValues: []*pb.DataNode{
					convertToDataNode(v),
				},
			})
		}
	}

	a := make([]*pb.WriteMetricsRequestV4_TaskResult, 0, len(taskResultByValueName))
	for _, r := range taskResultByValueName {
		a = append(a, r)
	}
	return a
}

func convertToDataNode(v interface{}) *pb.DataNode {
	switch x := v.(type) {
	case string:
		return &pb.DataNode{
			Type:  2,
			Count: 0,
			Bytes: []byte(x),
		}
	case float64:
		return &pb.DataNode{
			Type:  0,
			Count: 0,
			Value: x,
		}
	case *storage.AggNumberDataNode:
		if x.Agg == agg.AggAvg {
			var v float64
			if x.Count > 0 {
				v = x.Value / float64(x.Count)
			}
			return &pb.DataNode{
				Type:  1,
				Count: x.Count,
				Value: v,
			}
		} else {
			return &pb.DataNode{
				Type:  0,
				Count: x.Count,
				Value: x.Value,
			}
		}
	default:
		f64, _ := cast.ToFloat64E(x)
		return &pb.DataNode{
			Type:  0,
			Count: 0,
			Value: f64,
		}
	}
}

func convertToTaskResult(configKey, targetKey, metricName string, array []*model.DetailData) *pb.WriteMetricsRequestV4_TaskResult {

	a0 := array[0]
	tagKeys := make([]string, 0, len(a0.Tags))
	valueKeys := make([]string, 0, len(a0.Values))

	for k := range a0.Tags {
		tagKeys = append(tagKeys, k)
	}
	for k := range a0.Values {
		valueKeys = append(valueKeys, k)
	}

	tr := &pb.WriteMetricsRequestV4_TaskResult{
		Key:           configKey + "/" + targetKey,
		RefCollectKey: configKey,
		RefTargetKey:  targetKey,
		Table: &pb.WriteMetricsRequestV4_Table{
			Header: &pb.WriteMetricsRequestV4_Header{
				MetricName: metricName,
				TagKeys:    tagKeys,
				ValueKeys:  valueKeys,
			},
		},
	}

rowLoop:
	for _, a := range array {
		if len(a0.Tags) != len(a.Tags) {
			continue
		}
		if len(a0.Values) != len(a.Values) {
			continue
		}

		// TODO 这里的实现比较简单, 每个DetailData2自己就是一个 TaskResult, 其实是可以合并的
		tagValues := make([]string, len(a0.Tags))
		valueValues := make([]*pb.DataNode, len(a0.Values))

		for i, key := range tagKeys {
			v, ok := a.Tags[key]
			if !ok {
				continue rowLoop
			}
			tagValues[i] = v
		}

		for i, key := range valueKeys {
			v, ok := a.Values[key]
			if !ok {
				continue rowLoop
			}
			switch x := v.(type) {
			case string:
				valueValues[i] = &pb.DataNode{
					// TODO 消灭硬编码
					Type:  2,
					Bytes: []byte(x),
				}
			default:
				valueValues[i] = &pb.DataNode{
					Type:  0,
					Value: cast.ToFloat64(x),
				}
			}
		}

		tr.Table.Rows = append(tr.Table.Rows, &pb.WriteMetricsRequestV4_Row{
			Timestamp:   a.Timestamp,
			TagValues:   tagValues,
			ValueValues: valueValues,
		})
	}

	return tr
}
