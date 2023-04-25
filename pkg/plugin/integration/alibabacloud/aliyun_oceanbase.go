/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

const (
	// https://help.aliyun.com/document_detail/201596.htm?spm=a2c4g.11186623.0.0.68a41153GfPlBj#concept-2038833
	// ob的指标非常 多/乱, 有过滤的我用代码生成, 没规律的手动填写
	// 由于指标非常 多/乱, 可能存在指标遗漏/错误, 这些只能上线之后遇到反馈了再去支持
	OceanbaseNamespace = "acs_oceanbase"
	OceanbasePefix     = "alibabacloud_oceanbase"
)

var (
	clusterDimensions  = []string{"obClusterId"}
	tenantDimensions   = []string{"obClusterId", "obTenantId"}
	instanceDimensions = []string{"obClusterId", "obInstanceId"}

	OceanbaseMetricNames = []*AliyunMetricDesc{
		// 这些是不规则的metrics
		{MetricName: "disk_ob_data_usage", AliyunMetricName: "disk_ob_data_usage", Max: true, Dimensions: clusterDimensions},
		{MetricName: "disk_ob_data_size_instance", AliyunMetricName: "disk_ob_data_size_instance", Average: true, Dimensions: instanceDimensions},
		{MetricName: "disk_ob_data_usage_instance", AliyunMetricName: "disk_ob_data_usage_instance", Average: true, Dimensions: instanceDimensions},
		{MetricName: "load_load1_instance", AliyunMetricName: "load_load1_instance", Average: true, Dimensions: instanceDimensions},
	}
)

func addOb(metric string, hasTenant, hasInstance bool) {
	OceanbaseMetricNames = append(OceanbaseMetricNames, &AliyunMetricDesc{MetricName: metric, AliyunMetricName: metric, Dimensions: clusterDimensions, Sum: true})

	if hasTenant {
		tenantMetric := metric + "_tenant"
		OceanbaseMetricNames = append(OceanbaseMetricNames, &AliyunMetricDesc{MetricName: tenantMetric, AliyunMetricName: tenantMetric, Dimensions: tenantDimensions, Sum: true})
	}

	if hasInstance {
		instanceMetric := metric + "_instance"
		OceanbaseMetricNames = append(OceanbaseMetricNames, &AliyunMetricDesc{MetricName: instanceMetric, AliyunMetricName: instanceMetric, Dimensions: instanceDimensions, Average: true})
	}
}

func addObAvg(metric string) {
	addObExt(metric, "avg", "avg", "avg")
}

func addObRt(metric string, hasTenant, hasInstance bool) {
	addObExt(metric, "avg", "max", "avg")
}

func addObExt(metric string, clusterAgg, tenantAgg, instanceAgg string) {
	if clusterAgg != "" {
		OceanbaseMetricNames = append(OceanbaseMetricNames, setAgg(&AliyunMetricDesc{MetricName: metric, AliyunMetricName: metric, Dimensions: clusterDimensions}, clusterAgg))
	}

	if tenantAgg != "" {
		tenantMetric := metric + "_tenant"
		OceanbaseMetricNames = append(OceanbaseMetricNames, setAgg(&AliyunMetricDesc{MetricName: tenantMetric, AliyunMetricName: tenantMetric, Dimensions: tenantDimensions}, tenantAgg))
	}

	if instanceAgg != "" {
		instanceMetric := metric + "_instance"
		OceanbaseMetricNames = append(OceanbaseMetricNames, setAgg(&AliyunMetricDesc{MetricName: instanceMetric, AliyunMetricName: instanceMetric, Dimensions: instanceDimensions}, instanceAgg))
	}
}

func setAgg(desc *AliyunMetricDesc, agg string) *AliyunMetricDesc {
	switch agg {
	case "sum":
		desc.Sum = true
	case "avg":
		desc.Average = true
	case "max":
		desc.Max = true
	}
	return desc
}

func init() {

	addOb("active_sessions", true, true)
	addObAvg("request_queue_rt")

	// io数量
	addOb("io_read_count", true, true)
	addOb("io_write_count", true, true)

	// io bytes
	addOb("io_read_bytes", true, true)
	addOb("io_write_bytes", true, true)

	// io 延迟
	addOb("io_read_delay", true, true)
	addOb("io_write_delay", true, true)

	// io耗时
	addObAvg("io_read_rt")
	addObAvg("io_write_rt")

	// 各种sql数量统计
	addOb("sql_insert_count", true, true)
	addOb("sql_delete_count", true, true)
	addOb("sql_replace_count", true, true)
	addOb("sql_update_count", true, true)
	addOb("sql_select_count", true, true)

	// 各种sql耗时
	addObAvg("sql_insert_rt")
	addObAvg("sql_delete_rt")
	addObAvg("sql_update_rt")
	addObAvg("sql_replace_rt")
	addObAvg("sql_select_rt")

	// 各种存储数量统计
	addOb("storage_insert_row_count", true, true)
	// 这里确实没有tenant级的
	addOb("storage_read_row_count", false, true)
	addOb("storage_update_row_count", true, true)
	addOb("storage_delete_row_count", true, true)

	// qps
	addOb("qps", true, true)
	addObAvg("qps_rt")

	// 事务tps
	addOb("tps", true, true)
	addObRt("tps_rt", true, true)

	// 事务提交量
	addOb("trans_commit_count", true, true)
	addObRt("trans_commit_rt", true, true)

	// 事务回滚量
	addOb("trans_rollback_count", true, true)
	addObAvg("trans_rollback_rt")

	// 事务超时量
	addOb("trans_timeout_count", true, true)
}
