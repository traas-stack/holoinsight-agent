package alibabacloud

const (
	RedisNamespace = "acs_kvstore"
	RedisPrefix    = "alibabacloud_redis"
)

var (
	// redis 标准版
	RedisStandardMetricNames = []*AliyunMetricDesc{
		// 平均响应时间
		{MetricName: "standard_avg_rt", AliyunMetricName: "StandardAvgRt", Average: true, Maximum: true},
		// 连接数使用率
		{MetricName: "standard_connection_usage", AliyunMetricName: "StandardConnectionUsage", Average: true, Maximum: true},
		// CPU使用率
		{MetricName: "standard_cpu_usage", AliyunMetricName: "StandardCpuUsage", Average: true, Maximum: true},
		// 命令失败次数
		{MetricName: "standard_failed_count", AliyunMetricName: "StandardFailedCount", Average: true, Maximum: true},
		// 命中率
		{MetricName: "standard_hit_rate", AliyunMetricName: "StandardHitRate", Average: true, Maximum: true},
		// 入方向流量
		{MetricName: "standard_intranet_in", AliyunMetricName: "StandardIntranetIn", Average: true, Maximum: true},
		// 流入带宽使用率
		{MetricName: "standard_intranet_in_ratio", AliyunMetricName: "StandardIntranetInRatio", Average: true, Maximum: true},
		// 出方向流量
		{MetricName: "standard_intranet_out", AliyunMetricName: "StandardIntranetOut", Average: true, Maximum: true},
		// 流出带宽使用率
		{MetricName: "standard_intranet_out_ratio", AliyunMetricName: "StandardIntranetOutRatio", Average: true, Maximum: true},
		// 缓存内Key数量
		{MetricName: "standard_keys", AliyunMetricName: "StandardKeys", Average: true, Maximum: true},
		// 最大响应时间
		{MetricName: "standard_max_rt", AliyunMetricName: "StandardMaxRt", Average: true, Maximum: true},
		// 内存使用率
		{MetricName: "standard_memory_usage", AliyunMetricName: "StandardMemoryUsage", Average: true, Maximum: true},
		// QPS使用率
		{MetricName: "standard_qps_usage", AliyunMetricName: "StandardQPSUsage", Average: true, Maximum: true},
		// 数据同步时延
		{MetricName: "standard_sync_delay_time", AliyunMetricName: "StandardSyncDelayTime", Average: true, Maximum: true},
		// 已用连接数
		{MetricName: "standard_used_connection", AliyunMetricName: "StandardUsedConnection", Average: true, Maximum: true},
		// 内存使用量
		{MetricName: "standard_used_memory", AliyunMetricName: "StandardUsedMemory", Average: true, Maximum: true},
		// 平均每秒访问次数
		{MetricName: "standard_used_qps", AliyunMetricName: "StandardUsedQPS", Average: true, Maximum: true},
	}
	// redis 集群版
	RedisClusterMetricNames = []*AliyunMetricDesc{
		// 平均响应时间
		{MetricName: "sharding_avg_rt", AliyunMetricName: "ShardingAvgRt", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 连接数使用率
		{MetricName: "sharding_connection_usage", AliyunMetricName: "ShardingConnectionUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// CPU使用率
		{MetricName: "sharding_cpu_usage", AliyunMetricName: "ShardingCpuUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 命中率
		{MetricName: "sharding_hit_rate", AliyunMetricName: "ShardingHitRate", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 入方向流量
		{MetricName: "sharding_intranet_in", AliyunMetricName: "ShardingIntranetIn", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 流入带宽使用率
		{MetricName: "sharding_intranet_in_ratio", AliyunMetricName: "ShardingIntranetInRatio", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 出方向流量
		{MetricName: "sharding_intranet_out", AliyunMetricName: "ShardingIntranetOut", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 流出带宽使用率
		{MetricName: "sharding_intranet_out_ratio", AliyunMetricName: "ShardingIntranetOutRatio", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 缓存内Key数量
		{MetricName: "sharding_keys", AliyunMetricName: "ShardingKeys", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 最大响应时间
		{MetricName: "sharding_max_rt", AliyunMetricName: "ShardingMaxRt", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 内存使用率
		{MetricName: "sharding_memory_usage", AliyunMetricName: "ShardingMemoryUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy单个请求的平均字节数
		{MetricName: "sharding_proxy_avg_request_size", AliyunMetricName: "ShardingProxyAvgRequestSize", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy单个响应的平均字节数
		{MetricName: "sharding_proxy_avg_response_size", AliyunMetricName: "ShardingProxyAvgResponseSize", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy平均时延
		{MetricName: "sharding_proxy_avg_rt", AliyunMetricName: "ShardingProxyAvgRt", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy连接数使用率
		{MetricName: "sharding_proxy_connection_usage", AliyunMetricName: "ShardingProxyConnectionUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy CPU使用率
		{MetricName: "sharding_proxy_cpu_usage", AliyunMetricName: "ShardingProxyCpuUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy入流量速率
		{MetricName: "sharding_proxy_intranet_in", AliyunMetricName: "ShardingProxyIntranetIn", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy出流量速率
		{MetricName: "sharding_proxy_intranet_out", AliyunMetricName: "ShardingProxyIntranetOut", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy单个请求最大字节数
		{MetricName: "sharding_proxy_max_request_size", AliyunMetricName: "ShardingProxyMaxRequestSize", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy单个响应的最大字节数
		{MetricName: "sharding_proxy_max_response_size", AliyunMetricName: "ShardingProxyMaxResponseSize", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy最大时延
		{MetricName: "sharding_proxy_max_rt", AliyunMetricName: "ShardingProxyMaxRt", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy每秒总请求数
		{MetricName: "sharding_proxy_total_qps", AliyunMetricName: "ShardingProxyTotalQps", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy已使用连接数
		{MetricName: "sharding_proxy_used_connection", AliyunMetricName: "ShardingProxyUsedConnection", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// QPS使用率
		{MetricName: "sharding_qps_usage", AliyunMetricName: "ShardingQPSUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 数据同步时延
		{MetricName: "sharding_sync_delay_time", AliyunMetricName: "ShardingSyncDelayTime", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 已用连接数
		{MetricName: "sharding_used_connection", AliyunMetricName: "ShardingUsedConnection", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 内存使用量
		{MetricName: "sharding_used_memory", AliyunMetricName: "ShardingUsedMemory", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 平均每秒访问次数
		{MetricName: "sharding_used_qps", AliyunMetricName: "ShardingUsedQPS", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
	}
	// redis 读写分离版
	RedisRwsplitMetricNames = []*AliyunMetricDesc{
		// 平均响应时间
		{MetricName: "splitrw_avg_rt", AliyunMetricName: "SplitrwAvgRt", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 连接数使用率
		{MetricName: "splitrw_connection_usage", AliyunMetricName: "SplitrwConnectionUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// CPU使用率
		{MetricName: "splitrw_cpu_usage", AliyunMetricName: "SplitrwCpuUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 命令失败次数
		{MetricName: "splitrw_failed_count", AliyunMetricName: "SplitrwFailedCount", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 命中率
		{MetricName: "splitrw_hit_rate", AliyunMetricName: "SplitrwHitRate", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 入方向流量
		{MetricName: "splitrw_intranet_in", AliyunMetricName: "SplitrwIntranetIn", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 流入带宽使用率
		{MetricName: "splitrw_intranet_in_ratio", AliyunMetricName: "SplitrwIntranetInRatio", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 出方向流量
		{MetricName: "splitrw_intranet_out", AliyunMetricName: "SplitrwIntranetOut", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 流出带宽使用率
		{MetricName: "splitrw_intranet_out_ratio", AliyunMetricName: "SplitrwIntranetOutRatio", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 缓存内 Key 数量
		{MetricName: "splitrw_keys", AliyunMetricName: "SplitrwKeys", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 最大响应时间
		{MetricName: "splitrw_max_rt", AliyunMetricName: "SplitrwMaxRt", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 内存使用率
		{MetricName: "splitrw_memory_usage", AliyunMetricName: "SplitrwMemoryUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy单个请求的平均字节数
		{MetricName: "splitrw_proxy_avg_request_size", AliyunMetricName: "SplitrwProxyAvgRequestSize", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy单个响应的平均字节数
		{MetricName: "splitrw_proxy_avg_response_size", AliyunMetricName: "SplitrwProxyAvgResponseSize", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy平均时延
		{MetricName: "splitrw_proxy_avg_rt", AliyunMetricName: "SplitrwProxyAvgRt", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy连接数使用率
		{MetricName: "splitrw_proxy_connection_usage", AliyunMetricName: "SplitrwProxyConnectionUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy CPU使用率
		{MetricName: "splitrw_proxy_cpu_usage", AliyunMetricName: "SplitrwProxyCpuUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy入流量速率
		{MetricName: "splitrw_proxy_intranet_in", AliyunMetricName: "SplitrwProxyIntranetIn", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy出流量速率
		{MetricName: "splitrw_proxy_intranet_out", AliyunMetricName: "SplitrwProxyIntranetOut", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy单个请求最大字节数
		{MetricName: "splitrw_proxy_max_request_size", AliyunMetricName: "SplitrwProxyMaxRequestSize", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy单个响应的最大字节数
		{MetricName: "splitrw_proxy_max_response_size", AliyunMetricName: "SplitrwProxyMaxResponseSize", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy最大时延
		{MetricName: "splitrw_proxy_max_rt", AliyunMetricName: "SplitrwProxyMaxRt", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// Proxy每秒总请求数
		{MetricName: "splitrw_proxy_total_qps", AliyunMetricName: "SplitrwProxyTotalQps", Average: true, Maximum: true},
		// Proxy已使用连接数
		{MetricName: "splitrw_proxy_used_connection", AliyunMetricName: "SplitrwProxyUsedConnection", Average: true, Maximum: true},
		// QPS使用率
		{MetricName: "splitrw_qps_usage", AliyunMetricName: "SplitrwQPSUsage", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 数据同步时延
		{MetricName: "splitrw_sync_delay_time", AliyunMetricName: "SplitrwSyncDelayTime", Average: true, Maximum: true},
		// 已用连接数
		{MetricName: "splitrw_used_connection", AliyunMetricName: "SplitrwUsedConnection", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 内存使用量
		{MetricName: "splitrw_used_memory", AliyunMetricName: "SplitrwUsedMemory", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
		// 平均每秒访问次数
		{MetricName: "splitrw_used_qps", AliyunMetricName: "SplitrwUsedQPS", Dimensions: []string{"nodeId"}, Average: true, Maximum: true},
	}
)
