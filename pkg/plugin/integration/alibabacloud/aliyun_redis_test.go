/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

import (
	"fmt"
	"testing"
)

func TestRedisStandard(t *testing.T) {
	//ak := ""
	//sk := ""

	// 我们选择与我们机器近的endpoint即可
	//ams := NewAliyunMetricsService("metrics.cn-hangzhou.aliyuncs.com")

	//fmt.Println("ecs")
	//ams.Dump(ak, sk, EcsNamespace, EcsPrefix, EcsMetricNames)
	//
	//fmt.Println("rds")
	//ams.Dump(ak, sk, RdsNamespace, RdsPrefix, RdsMetricNames)
	//
	//fmt.Println("redis-standard")
	//ams.Dump(ak, sk, RedisNamespace, RedisPrefix, RedisStandardMetricNames)
	//
	//fmt.Println("redis-cluster")
	//ams.Dump(ak, sk, RedisNamespace, RedisPrefix, RedisClusterMetricNames)

	fmt.Println("redis-rwsplit")
	// ams.Dump(ak, sk, RedisNamespace, RedisPrefix, RedisRwsplitMetricNames)

	fmt.Println("slb")
	// ams.Dump(ak, sk, SlbNamespace, SlbPrefix, SlbMetricNames)

	//
	// fmt.Println("alb")
	// ams.Dump(ak, sk, AlbNamespace, AlbPrefix, AlbMetricNames)

}
