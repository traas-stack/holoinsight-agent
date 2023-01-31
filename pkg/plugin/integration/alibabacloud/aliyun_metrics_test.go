package alibabacloud

import (
	"fmt"
	client2 "github.com/alibabacloud-go/cms-20190101/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"testing"
	"time"
)

func TestAliyunOpenAPI(t *testing.T) {
	ams := NewAliyunMetricsService()
	ak := ""
	sk := ""
	endpoint := "metrics.ap-northeast-1.aliyuncs.com"
	metrics, err := ams.DescribeMetricList2(nil, &DescribeMetricList2Request{
		Endpoint:  endpoint,
		Start:     time.Now().Truncate(time.Minute).Add(-5 * time.Minute),
		End:       time.Now().Truncate(time.Minute),
		Namespace: "acs_tair",
		Name:      "TairPDBStandardAvgIORt",
		Ak:        ak,
		Sk:        sk,
	})
	fmt.Println(err)
	for _, metric := range metrics {
		fmt.Println(metric.Timestamp, metric.Name, metric.Tags, metric.Value)
	}

	c, _ := ams.GetClient(endpoint, ak, sk)
	resp, err := c.DescribeMetricMetaList(&client2.DescribeMetricMetaListRequest{
		//Labels:     nil,
		//MetricName: tea.String("MeteringSyncRX"),
		//Namespace:  tea.String("acs_oss_dashboard"),
		//PageNumber: nil,
		//PageSize:   nil,
		//RegionId:   tea.String("cn-hongkong"),
		PageNumber: tea.Int32(1),
		PageSize:   tea.Int32(1),
	})
	//resp.Body.Resources.Resource[0].Periods
	fmt.Println(resp)
	//for _, r := range resp.Body.Resources.Resource {
	//	//r.Namespace
	//	//r.MetricName
	//	//r.Periods
	//}

	//fmt.Println(c.DescribeHybridMonitorNamespaceList(&client2.DescribeHybridMonitorNamespaceListRequest{
	//	Keyword:           nil,
	//	Namespace:         nil,
	//	PageNumber:        tea.Int32(1),
	//	PageSize:          tea.Int32(100),
	//	RegionId:          nil,
	//	ShowTaskStatistic: nil,
	//}))
}
