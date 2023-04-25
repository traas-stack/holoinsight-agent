/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

const (
	// 官方已经不推荐使用了
	// https://help.aliyun.com/document_detail/162852.html
	SlbNamespace = "acs_slb_dashboard"
	SlbPrefix    = "alibabacloud_slb"
)

var (
	SlbMetricNames = []*AliyunMetricDesc{
		// "监听"是细到端口粒度
		// TCP每秒活跃连接数
		{MetricName: "active_connection", AliyunMetricName: "ActiveConnection", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 监听每秒丢失连接数
		{MetricName: "drop_connection", AliyunMetricName: "DropConnection", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 监听每秒丢失入包数
		{MetricName: "drop_packet_rx", AliyunMetricName: "DropPacketRX", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 监听每秒丢失出包数
		{MetricName: "drop_packet_tx", AliyunMetricName: "DropPacketTX", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 监听每秒丢失入bit数
		{MetricName: "drop_traffic_rx", AliyunMetricName: "DropTrafficRX", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 监听每秒丢失出bit数
		{MetricName: "drop_traffic_tx", AliyunMetricName: "DropTrafficTX", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// （分组维度）TCP每秒活跃连接数
		{MetricName: "group_active_connection", AliyunMetricName: "GroupActiveConnection", Dimensions: []string{"groupId"}, Average: true, Minimum: true, Maximum: true},
		// （分组维度）TCP新建连接数
		{MetricName: "group_new_connection", AliyunMetricName: "GroupNewConnection", Dimensions: []string{"groupId"}, Average: true, Minimum: true, Maximum: true},
		// TotalTrafficRX
		{MetricName: "group_total_traffic_rx", AliyunMetricName: "GroupTotalTrafficRX", Dimensions: []string{"groupId"}, Value: true},
		// TotalTrafficTX
		{MetricName: "group_total_traffic_tx", AliyunMetricName: "GroupTotalTrafficTX", Dimensions: []string{"groupId"}, Value: true},
		// （分组维度）流入带宽
		{MetricName: "group_traffic_rx", AliyunMetricName: "GroupTrafficRX", Dimensions: []string{"groupId"}, Average: true, Minimum: true, Maximum: true},
		// （分组维度）流出带宽
		{MetricName: "group_traffic_tx", AliyunMetricName: "GroupTrafficTX", Dimensions: []string{"groupId"}, Average: true, Minimum: true, Maximum: true},
		// （分组维度）后端异常ECS实例个数
		{MetricName: "group_unhealthy_server_count", AliyunMetricName: "GroupUnhealthyServerCount", Dimensions: []string{"groupId"}, Average: true, Minimum: true, Maximum: true},
		// 七层转发规则后端健康ECS实例个数
		{MetricName: "healthy_server_count_with_rule", AliyunMetricName: "HealthyServerCountWithRule", Dimensions: []string{"port", "vip", "domain", "url"}, Average: true, Minimum: true, Maximum: true},
		// 健康检查后端健康ECS实例个数
		{MetricName: "heathy_server_count", AliyunMetricName: "HeathyServerCount", Dimensions: []string{"port", "vip"}, Average: true, Minimum: true, Maximum: true},
		// TCP非活跃连接数
		{MetricName: "inactive_connection", AliyunMetricName: "InactiveConnection", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 实例每秒活跃连接数
		{MetricName: "instance_active_connection", AliyunMetricName: "InstanceActiveConnection", Average: true, Minimum: true, Maximum: true},
		// 实例每秒丢失连接数
		{MetricName: "instance_drop_connection", AliyunMetricName: "InstanceDropConnection", Average: true, Minimum: true, Maximum: true},
		// 实例每秒丢失入包数
		{MetricName: "instance_drop_packet_rx", AliyunMetricName: "InstanceDropPacketRX", Average: true, Minimum: true, Maximum: true},
		// 实例每秒丢失出包数
		{MetricName: "instance_drop_packet_tx", AliyunMetricName: "InstanceDropPacketTX", Average: true, Minimum: true, Maximum: true},
		// 实例每秒丢失入bit数
		{MetricName: "instance_drop_traffic_rx", AliyunMetricName: "InstanceDropTrafficRX", Average: true, Minimum: true, Maximum: true},
		// 实例每秒丢失出bit数
		{MetricName: "instance_drop_traffic_tx", AliyunMetricName: "InstanceDropTrafficTX", Average: true, Minimum: true, Maximum: true},
		// 实例每秒非活跃连接数
		{MetricName: "instance_inactive_connection", AliyunMetricName: "InstanceInactiveConnection", Average: true, Minimum: true, Maximum: true},
		// 实例每秒最大并发连接数
		{MetricName: "instance_max_connection", AliyunMetricName: "InstanceMaxConnection", Average: true, Minimum: true, Maximum: true},
		// 实例最大连接数使用率
		{MetricName: "instance_max_connection_utilization", AliyunMetricName: "InstanceMaxConnectionUtilization", Average: true, Minimum: true, Maximum: true},
		// 实例每秒新建连接数
		{MetricName: "instance_new_connection", AliyunMetricName: "InstanceNewConnection", Average: true, Minimum: true, Maximum: true},
		// 实例新建连接数使用率
		{MetricName: "instance_new_connection_utilization", AliyunMetricName: "InstanceNewConnectionUtilization", Average: true, Minimum: true, Maximum: true},
		// 实例每秒入包数
		{MetricName: "instance_packet_rx", AliyunMetricName: "InstancePacketRX", Average: true, Minimum: true, Maximum: true},
		// 实例每秒出包数
		{MetricName: "instance_packet_tx", AliyunMetricName: "InstancePacketTX", Average: true, Minimum: true, Maximum: true},
		// 七层实例QPS
		{MetricName: "instance_qps", AliyunMetricName: "InstanceQps", Average: true},
		// 七层实例QPS使用率
		{MetricName: "instance_qps_utilization", AliyunMetricName: "InstanceQpsUtilization", Average: true, Minimum: true, Maximum: true},
		// 七层实例RT
		{MetricName: "instance_rt", AliyunMetricName: "InstanceRt", Average: true},
		// 七层实例每秒状态码2XX数量
		{MetricName: "instance_status_code_2xx", AliyunMetricName: "InstanceStatusCode2xx", Average: true},
		// 七层实例每秒状态码3XX数量
		{MetricName: "instance_status_code_3xx", AliyunMetricName: "InstanceStatusCode3xx", Average: true},
		// 七层实例每秒状态码4XX数量
		{MetricName: "instance_status_code_4xx", AliyunMetricName: "InstanceStatusCode4xx", Average: true},
		// 七层实例每秒状态码5XX数量
		{MetricName: "instance_status_code_5xx", AliyunMetricName: "InstanceStatusCode5xx", Average: true},
		// 七层实例每秒其它状态码数量
		{MetricName: "instance_status_code_other", AliyunMetricName: "InstanceStatusCodeOther", Average: true},
		// 实例每秒入bit数
		{MetricName: "instance_traffic_rx", AliyunMetricName: "InstanceTrafficRX", Average: true, Minimum: true, Maximum: true},
		// 实例每秒出bit数
		{MetricName: "instance_traffic_tx", AliyunMetricName: "InstanceTrafficTX", Average: true, Minimum: true, Maximum: true},
		// 实例网络流出带宽使用率
		{MetricName: "instance_traffic_tx_utilization", AliyunMetricName: "InstanceTrafficTXUtilization", Average: true, Minimum: true, Maximum: true},
		// 七层实例每秒UpstreamCode4xx数量
		{MetricName: "instance_upstream_code_4xx", AliyunMetricName: "InstanceUpstreamCode4xx", Average: true},
		// 七层实例每秒UpstreamCode5xx数量
		{MetricName: "instance_upstream_code_5xx", AliyunMetricName: "InstanceUpstreamCode5xx", Average: true},
		// 七层实例UpstreamRt
		{MetricName: "instance_upstream_rt", AliyunMetricName: "InstanceUpstreamRt", Average: true},
		// 端口并发连接数
		{MetricName: "max_connection", AliyunMetricName: "MaxConnection", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// TCP新建连接数
		{MetricName: "new_connection", AliyunMetricName: "NewConnection", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 每秒流入数据包数
		{MetricName: "packet_rx", AliyunMetricName: "PacketRX", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 每秒流出数据包数
		{MetricName: "packet_tx", AliyunMetricName: "PacketTX", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 七层监听QPS
		{MetricName: "qps", AliyunMetricName: "Qps", Dimensions: []string{"port", "protocol"}, Average: true},
		// 七层监听RT
		{MetricName: "rt", AliyunMetricName: "Rt", Dimensions: []string{"port", "protocol"}, Average: true},
		// 七层监听每秒状态码2XX数量
		{MetricName: "status_code_2xx", AliyunMetricName: "StatusCode2xx", Dimensions: []string{"port", "protocol"}, Average: true},
		// 七层监听每秒状态码3XX数量
		{MetricName: "status_code_3xx", AliyunMetricName: "StatusCode3xx", Dimensions: []string{"port", "protocol"}, Average: true},
		// 七层监听每秒状态码4XX数量
		{MetricName: "status_code_4xx", AliyunMetricName: "StatusCode4xx", Dimensions: []string{"port", "protocol"}, Average: true},
		// 七层监听每秒状态码5XX数量
		{MetricName: "status_code_5xx", AliyunMetricName: "StatusCode5xx", Dimensions: []string{"port", "protocol"}, Average: true},
		// 七层监听每秒其它状态码数量
		{MetricName: "status_code_other", AliyunMetricName: "StatusCodeOther", Dimensions: []string{"port", "protocol"}, Average: true},
		// 流入带宽
		{MetricName: "traffic_rx_new", AliyunMetricName: "TrafficRXNew", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 流出带宽
		{MetricName: "traffic_tx_new", AliyunMetricName: "TrafficTXNew", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 后端异常ECS实例个数
		{MetricName: "unhealthy_server_count", AliyunMetricName: "UnhealthyServerCount", Dimensions: []string{"port", "vip"}, Average: true, Minimum: true, Maximum: true},
		// 七层转发规则后端异常ECS实例个数
		{MetricName: "unhealthy_server_count_with_rule", AliyunMetricName: "UnhealthyServerCountWithRule", Dimensions: []string{"port", "vip", "domain", "url"}, Average: true, Minimum: true, Maximum: true},
		// 七层监听每秒UpstreamCode4xx数量
		{MetricName: "upstream_code_4xx", AliyunMetricName: "UpstreamCode4xx", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 七层监听每秒UpstreamCode5xx数量
		{MetricName: "upstream_code_5xx", AliyunMetricName: "UpstreamCode5xx", Dimensions: []string{"port", "protocol"}, Average: true, Minimum: true, Maximum: true},
		// 七层监听UpstreamRt
		{MetricName: "upstream_rt", AliyunMetricName: "UpstreamRt", Dimensions: []string{"port", "protocol"}, Average: true},
	}
)
