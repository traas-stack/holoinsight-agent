package gateway

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util/stat"
	"strings"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
	"github.com/traas-stack/holoinsight-agent/pkg/util/batch"
	"github.com/spf13/cast"
)

const (
	defaultTimeout = 5 * time.Second
)

type (
	gatewayOutput struct {
		processor batch.Processor
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
	pI, err := gatewayProcessorSingletonHolder.Acquire()
	if err != nil {
		return nil, err
	}
	processor := pI.(batch.Processor)
	return &gatewayOutput{
		processor: processor,
	}, nil
}

func (c *gatewayOutput) Start() {

}

func (c *gatewayOutput) Stop() {
	gatewayProcessorSingletonHolder.Release()
}

func (c *gatewayOutput) WriteBatchAsync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	go c.WriteBatchSync(configKey, targetKey, metricName, array)
	return nil
}

var outputStat = stat.DefaultManager1S.Counter("output.gateway")

func (c *gatewayOutput) WriteBatchSync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	// TODO 要在这里做攒批吗?

	// TODO 大部分场景 异步化 不要阻塞 但失败了要记录日志
	// TODO 对于重要指标 要同步化 如果返回了可重试的失败则立即重试 并记录日志

	// c.processor.Put(convertToTaskResult(configKey, targetKey, metricName, array))
	converted := convertToTaskResult2(configKey, targetKey, metricName, array)
	outputStat.Add([]string{
		configKey,
	}, []int64{
		int64(len(converted)),
	})
	for _, tr := range converted {
		c.processor.Put(tr)
	}

	// 	c.processor.Put(convertToTaskResult(configKey, targetKey, metricName, array))

	return nil
}

var singleValueKeys = []string{"value"}

// TODO 解释一下
func convertToTaskResult2(configKey, targetKey, metricName string, array []*model.DetailData) []*pb.WriteMetricsRequestV4_TaskResult {
	var a []*pb.WriteMetricsRequestV4_TaskResult
	for _, i := range array {
		tagKeys := make([]string, 0, len(i.Tags))
		tagValues := make([]string, 0, len(i.Tags))
		for k, v := range i.Tags {
			tagKeys = append(tagKeys, k)
			tagValues = append(tagValues, v)
		}
		if i.SingleValue {
			for _, v := range i.Values {
				a = append(a, &pb.WriteMetricsRequestV4_TaskResult{
					Key:           configKey + "/" + targetKey,
					RefCollectKey: configKey,
					RefTargetKey:  targetKey,
					Table: &pb.WriteMetricsRequestV4_Table{
						Header: &pb.WriteMetricsRequestV4_Header{
							MetricName: metricName,
							TagKeys:    tagKeys,
							ValueKeys:  singleValueKeys,
						},
						Rows: []*pb.WriteMetricsRequestV4_Row{
							{
								Timestamp: i.Timestamp,
								TagValues: tagValues,
								ValueValues: []*pb.DataNode{
									convertToDataNode(v),
								},
							},
						},
					},
				})
			}
			continue
		}
		for vk, v := range i.Values {
			name := vk
			if metricName != "" {
				if strings.Contains(metricName, "%s") {
					name = strings.Replace(metricName, "%s", vk, 1)
				} else if strings.HasSuffix(metricName, ".") || strings.HasSuffix(metricName, "_") {
					name = metricName + vk
				} else {
					name = metricName + "_" + vk
				}
			}
			a = append(a, &pb.WriteMetricsRequestV4_TaskResult{
				Key:           configKey + "/" + targetKey,
				RefCollectKey: configKey,
				RefTargetKey:  targetKey,
				Table: &pb.WriteMetricsRequestV4_Table{
					Header: &pb.WriteMetricsRequestV4_Header{
						MetricName: name,
						TagKeys:    tagKeys,
						ValueKeys:  singleValueKeys,
					},
					Rows: []*pb.WriteMetricsRequestV4_Row{
						{
							Timestamp: i.Timestamp,
							TagValues: tagValues,
							ValueValues: []*pb.DataNode{
								convertToDataNode(v),
							},
						},
					},
				},
			})
		}
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
