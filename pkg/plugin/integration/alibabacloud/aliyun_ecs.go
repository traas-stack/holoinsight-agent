/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

const (
	// https://help.aliyun.com/document_detail/162844.html?spm=a2c4g.11186623.0.0.4b776c8aOiSc8z
	EcsNamespace = "acs_ecs_dashboard"
	EcsPrefix    = "alibabacloud_ecs"
)

var (
	EcsMetricNames = []*AliyunMetricDesc{
		// （ECS）CPU使用率 %
		{MetricName: "cpu_utilization", AliyunMetricName: "CPUUtilization", Average: true},

		// （ECS）所有磁盘读取BPS Byte/s
		{MetricName: "disk_read_bps", AliyunMetricName: "DiskReadBPS", Average: true},
		// （ECS）所有磁盘每秒读取次数 Count/s
		{MetricName: "disk_read_iops", AliyunMetricName: "DiskReadIOPS", Average: true},
		// （ECS）所有磁盘写入BPS Byte/s
		{MetricName: "disk_write_bps", AliyunMetricName: "DiskWriteBPS", Average: true},
		// （ECS）所有磁盘每秒写入次数 Count/s
		{MetricName: "disk_write_iops", AliyunMetricName: "DiskWriteIOPS", Average: true},

		//（ECS）经典网络公网流入带宽
		{MetricName: "internet_in_rate", AliyunMetricName: "InternetInRate", Average: true},
		// （ECS）经典网络公网流出带宽
		{MetricName: "internet_out_rate", AliyunMetricName: "InternetOutRate", Average: true},
		// （ECS）经典网络公网流出带宽使用率
		{MetricName: "internet_out_rate_percent", AliyunMetricName: "InternetOutRate_Percent", Average: true},

		//（ECS）内网流入带宽 Bit/s
		{MetricName: "intranet_in_rate", AliyunMetricName: "IntranetInRate", Average: true},
		//（ECS）内网流出带宽 Bit/s
		{MetricName: "intranet_out_rate", AliyunMetricName: "IntranetOut", Average: true},

		// ECS）同时连接数
		{MetricName: "concurrent_connections", AliyunMetricName: "concurrentConnections", Maximum: true},

		// 以下指标依赖 阿里云agent, 必须要装了才会有数据
		// （Agent）cpu.cores
		{MetricName: "cpu_cores", AliyunMetricName: "cpu_cores", Value: true},
		// %
		{MetricName: "cpu_idle", AliyunMetricName: "cpu_idle", Average: true},
		{MetricName: "cpu_other", AliyunMetricName: "cpu_other", Average: true},
		{MetricName: "cpu_system", AliyunMetricName: "cpu_system", Average: true},
		{MetricName: "cpu_total", AliyunMetricName: "cpu_total", Average: true},
		{MetricName: "cpu_user", AliyunMetricName: "cpu_user", Average: true},
		{MetricName: "cpu_wait", AliyunMetricName: "cpu_wait", Average: true},

		// Byte/s
		{MetricName: "disk_readbytes", AliyunMetricName: "disk_readbytes", Average: true},
		// Count/s
		{MetricName: "disk_readiops", AliyunMetricName: "disk_readiops", Average: true},
		// Byte/s
		{MetricName: "disk_writebytes", AliyunMetricName: "disk_writebytes", Average: true},
		// Count/s
		{MetricName: "disk_writeiops", AliyunMetricName: "disk_writeiops", Average: true},
		// Byte 磁盘剩余空间
		{MetricName: "diskusage_avail", AliyunMetricName: "diskusage_avail", Average: true},

		{MetricName: "load_1m", AliyunMetricName: "load_1m", Average: true},
		{MetricName: "load_5m", AliyunMetricName: "load_5m", Average: true},
		{MetricName: "load_15m", AliyunMetricName: "load_15m", Average: true},

		// Byte
		{MetricName: "memory_totalspace", AliyunMetricName: "memory_totalspace", Average: true},
		// Byte
		{MetricName: "memory_usedspace", AliyunMetricName: "memory_usedspace", Average: true},
		// Byte
		{MetricName: "memory_freespace", AliyunMetricName: "memory_freespace", Average: true},
		// %
		{MetricName: "memory_freeutilization", AliyunMetricName: "memory_freeutilization", Average: true},
		// %
		{MetricName: "memory_usedutilization", AliyunMetricName: "memory_usedutilization", Average: true},

		// 以下暂时不用到
		// 进程粒度的 cpu 使用率
		// {MetricName: "process.cpu", AliyunMetricName: "process.cpu", Average: true},

		// 以下几个精确到 device 粒度, 现在用不到不用
		// {MetricName: "diskusage_free", AliyunMetricName: "diskusage_free", Average: true},
		// {MetricName: "diskusage_total", AliyunMetricName: "diskusage_total", Average: true},
		// {MetricName: "diskusage_used", AliyunMetricName: "diskusage_used", Average: true},
		// {MetricName: "diskusage_utilization", AliyunMetricName: "diskusage_utilization", Average: true},
	}
)
