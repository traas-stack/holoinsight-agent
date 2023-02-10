package alibabacloud

import (
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	cms20190101 "github.com/alibabacloud-go/cms-20190101/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"
	"strings"
	"sync"
)

type (
	metricMetaManager struct {
		periodCache sync.Map
	}
)

var metricMetaManagerInstance metricMetaManager

func (m *metricMetaManager) getPeriod(req *DescribeMetricList2Request) *string {
	key := req.Namespace + "/" + req.Name
	if p, ok := m.periodCache.Load(key); ok {
		return tea.String(p.(string))
	}
	return tea.String("60")
}

func (m *metricMetaManager) updatePeriodCache(request *DescribeMetricList2Request) error {
	req := &cms20190101.DescribeMetricMetaListRequest{
		Namespace:  tea.String(request.Namespace),
		MetricName: tea.String(request.Name),
	}
	client, err := clientManagerInstance.getClient(request.Endpoint, request.Ak, request.Sk)
	if err != nil {
		return err
	}
	resp, err := client.DescribeMetricMetaList(req)
	if err != nil {
		logger.Errorz("updatePeriodCache error", zap.Any("conf", request), zap.Error(err))
		return err
	}

	if len(resp.Body.Resources.Resource) == 1 {
		str := *resp.Body.Resources.Resource[0].Periods
		str = strings.Split(str, ",")[0]
		m.periodCache.Store(request.Namespace+"/"+request.Name, str)
	}
	return nil
}
