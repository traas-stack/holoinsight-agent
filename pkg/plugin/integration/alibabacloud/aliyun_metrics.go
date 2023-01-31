package alibabacloud

import (
	"encoding/json"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util/stat"
	cms20190101 "github.com/alibabacloud-go/cms-20190101/v7/client"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/spf13/cast"
	"go.uber.org/ratelimit"
	"strings"
	"time"
)

type (
	AliyunMetricsService struct {
	}
	DescribeMetricListResponse_Datapoint struct {
		Timestamp  int64   `json:"timestamp"`
		InstanceId string  `json:"instanceId"`
		UserId     string  `json:"userId"`
		Minimum    float64 `json:"Minimum"`
		Maximum    float64 `json:"Maximum"`
		Average    float64 `json:"Average"`
	}
)

var callStat = stat.DefaultManager.Counter("alibabacloud_metrics_openapi_callstat")
var clientManagerInstance clientManager

func init() {
	clientManagerInstance.start()
}

// see https://help.aliyun.com/document_detail/28616.html
func NewAliyunMetricsService() *AliyunMetricsService {
	return &AliyunMetricsService{}
}

func (s *AliyunMetricsService) DescribeMetricList(endpoint, ak, sk string, req *cms20190101.DescribeMetricListRequest) (*cms20190101.DescribeMetricListResponse, error) {
	client, err := clientManagerInstance.getClient(endpoint, ak, sk)
	if err != nil {
		return nil, err
	}
	runtime := &util.RuntimeOptions{}
	return client.DescribeMetricListWithOptions(req, runtime)
}

type (
	AliyunMetric struct {
		Name      string
		Tags      map[string]string
		Timestamp int64
		Value     map[string]float64
	}
	DescribeMetricList2Request struct {
		Start            time.Time
		End              time.Time
		Namespace        string
		Name             string
		Endpoint, Ak, Sk string
	}
)

func (s *AliyunMetricsService) GetClient(endpoint, ak, sk string) (*cms20190101.Client, error) {
	return clientManagerInstance.getClient(endpoint, ak, sk)
}

func (s *AliyunMetricsService) DescribeMetricList2(ratelimiter ratelimit.Limiter, request *DescribeMetricList2Request) ([]*AliyunMetric, error) {

	//if _, ok := invalidPeriodCache.Load(request.Namespace + "/" + request.Name); ok {
	//	return nil, errors.New("quick failure: invalid period")
	//}

	// 每次同步 [now-5min, now]的数据
	start := request.Start.Format("2006-01-02 15:04:05")
	end := request.End.Format("2006-01-02 15:04:05")

	var metrics []*AliyunMetric
	nextToken := ""

	periodUpdated := false
	for {
		req := &cms20190101.DescribeMetricListRequest{
			MetricName: tea.String(request.Name),
			Namespace:  tea.String(request.Namespace),
			Period:     metricMetaManagerInstance.getPeriod(request),
			StartTime:  tea.String(start),
			EndTime:    tea.String(end),
			// see https://help.aliyun.com/document_detail/51936.html
			Length:    tea.String("1440"),
			NextToken: tea.String(nextToken),
			// 测试用
			// Dimensions: tea.String("[{\"instanceId\":\"i-uf652exfq23hy5gqcf0q\"}]"),
		}
		if ratelimiter != nil {
			ratelimiter.Take()
		}
		queryBegin := time.Now()
		resp, err := s.DescribeMetricList(request.Endpoint, request.Ak, request.Sk, req)
		if err != nil {
			if !periodUpdated && strings.Contains(err.Error(), "must be divided evenly by one period") {
				periodUpdated = true
				if ratelimiter != nil {
					ratelimiter.Take()
				}
				if metricMetaManagerInstance.updatePeriodCache(request) == nil {
					continue
				}
			}
			return nil, err
		}
		queryEnd := time.Now()
		callStat.Add([]string{request.Ak}, []int64{1, queryEnd.Sub(queryBegin).Milliseconds()})

		if err != nil || resp == nil || resp.Body == nil {
			break
		}

		var dps []map[string]interface{}
		jsonBody := tea.StringValue(resp.Body.Datapoints)
		err = json.Unmarshal([]byte(jsonBody), &dps)
		if err != nil {
			return nil, err
		}
		for _, dp := range dps {

			m := &AliyunMetric{
				Name:      request.Name,
				Tags:      make(map[string]string),
				Timestamp: 0,
				Value:     make(map[string]float64),
			}

			for k, v := range dp {
				if k == "timestamp" {
					m.Timestamp = cast.ToInt64(v)
				} else {
					switch x := v.(type) {
					case string:
						m.Tags[k] = x
					default:
						m.Value[strings.ToLower(k)] = cast.ToFloat64(v)
					}
				}
			}

			metrics = append(metrics, m)
		}

		nextToken2 := tea.StringValue(resp.Body.NextToken)
		if nextToken2 == "" || nextToken == nextToken2 {
			break
		}
		nextToken = nextToken2
	}

	return metrics, nil
}
