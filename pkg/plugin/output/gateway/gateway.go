/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package gateway

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"go.uber.org/zap"
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

var (
	outputStat      = stat.DefaultManager1S.Counter("output.gateway")
	singleValueKeys = []string{"value"}
)

func (c *gatewayOutput) WriteMetricsV1(metrics []*model.Metric, oe output.Extension) {
	request := &WriteV1Request{
		Batch:     metrics,
		Extension: nil,
		NoMerge:   false,
	}

	request.Extension = make(map[string]string)
	if oe.Tenant != "" {
		request.Extension["tenant"] = oe.Tenant
		request.NoMerge = true
	}
	if oe.ConfigKey != "" {
		request.Extension["configKey"] = oe.ConfigKey
	}

	err := GetWriteService().WriteV1(context.Background(), request)
	if err != nil {
		logger.Errorz("[gateway] write error", zap.Error(err))
	}
}

func newGatewayOutput(config output.Config) (output.Output, error) {
	return &gatewayOutput{}, nil
}

func (c *gatewayOutput) Start() {

}

func (c *gatewayOutput) Stop() {
}

func (c *gatewayOutput) WriteBatchAsync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	batch := convertToTaskResult2(configKey, targetKey, metricName, array, nil)
	go GetWriteService().WriteV4(context.Background(), &WriteV4Request{Batch: batch})
	return nil
}

func (c *gatewayOutput) WriteBatchV4(configKey, targetKey, metricName string, array []*model.DetailData, completeness *output.PeriodCompleteness) error {
	batch := convertToTaskResult2(configKey, targetKey, metricName, array, completeness)
	return GetWriteService().WriteV4(context.Background(), &WriteV4Request{Batch: batch})
}

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

func convertToTaskResult2(configKey, targetKey, metricName string, array []*model.DetailData, completeness *output.PeriodCompleteness) []*pb.WriteMetricsRequestV4_TaskResult {
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

	a := make([]*pb.WriteMetricsRequestV4_TaskResult, 0, len(taskResultByValueName)+1)
	for _, r := range taskResultByValueName {
		r.Extension = map[string]string{
			"configKey": configKey,
		}
		a = append(a, r)
	}

	if completeness != nil {
		r := &pb.WriteMetricsRequestV4_TaskResult{
			Key:           configKey + "/" + targetKey,
			RefCollectKey: configKey,
			RefTargetKey:  targetKey,
			Table: &pb.WriteMetricsRequestV4_Table{
				Timestamp: completeness.TS,
			},
			Timestamp: completeness.TS,
			Completeness: &pb.WriteMetricsRequestV4_Completeness{
				Ok: completeness.OK,
			},
			Extension: map[string]string{
				"configKey": configKey,
			},
		}
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
