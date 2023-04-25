/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

import (
	"errors"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	rds20140815 "github.com/alibabacloud-go/rds-20140815/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"testing"
)

func TestRds2(t *testing.T) {
	ak := ""
	sk := ""
	//ams := NewAliyunMetricsService("metrics.cn-hangzhou.aliyuncs.com")
	region := "cn-hangzhou"
	config := &openapi.Config{
		AccessKeyId:     tea.String(ak),
		AccessKeySecret: tea.String(sk),
	}
	// 访问的域名
	config.Endpoint = tea.String("rds.aliyuncs.com")
	client, err := rds20140815.NewClient(config)
	if err != nil {
		panic(err)
	}

	resourceCount := 0
	nextToken := ""
	for {
		req := &rds20140815.DescribeDBInstancesRequest{
			RegionId: tea.String(region),
			PageSize: tea.Int32(100),
		}
		if nextToken != "" {
			req.NextToken = tea.String(nextToken)
		}
		resp, err := client.DescribeDBInstances(req)
		if err != nil {
			panic(err)
		}
		if resp == nil || resp.Body == nil {
			panic(errors.New("bad resp"))
		}
		resourceCount += len(resp.Body.Items.DBInstance)
		nextToken2 := tea.StringValue(resp.Body.NextToken)
		fmt.Println("nextToken", nextToken2)
		if nextToken == nextToken2 || nextToken2 == "" {
			break
		}
		nextToken = nextToken2
	}

	fmt.Printf("有 %d 台 RDS\n", resourceCount)

	if resourceCount == 0 {
		return
	}
	{
		//start := time.Now().Truncate(time.Minute).Add(-3 * time.Minute).Format("2006-01-02 15:04:05")
		//end := time.Now().Truncate(time.Minute).Add(-2 * time.Minute).Format("2006-01-02 15:04:05")
		// ams.Dump(ak, sk, RdsNamespace, RdsPrefix, RdsMetricNames)
	}
}
