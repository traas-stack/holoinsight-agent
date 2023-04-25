/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

// https://help.aliyun.com/document_detail/201639.html
const (
	AlbNamespace = "acs_alb"
	AlbPrefix    = "alibabacloud_alb"
)

var (
	AlbMetricNames = []*AliyunMetricDesc{
		// 监听活跃连接数
		{MetricName: "listener_active_connection", AliyunMetricName: "ListenerActiveConnection", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒TLS握手失败连接数
		{MetricName: "listener_client_tls_negotiation_error", AliyunMetricName: "ListenerClientTLSNegotiationError", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒2XX个数
		{MetricName: "listener_http_code_2xx", AliyunMetricName: "ListenerHTTPCode2XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒3XX个数
		{MetricName: "listener_http_code_3xx", AliyunMetricName: "ListenerHTTPCode3XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒4XX个数
		{MetricName: "listener_http_code_4xx", AliyunMetricName: "ListenerHTTPCode4XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒500个数
		{MetricName: "listener_http_code_500", AliyunMetricName: "ListenerHTTPCode500", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒502个数
		{MetricName: "listener_http_code_502", AliyunMetricName: "ListenerHTTPCode502", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒503个数
		{MetricName: "listener_http_code_503", AliyunMetricName: "ListenerHTTPCode503", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒504个数
		{MetricName: "listener_http_code_504", AliyunMetricName: "ListenerHTTPCode504", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒5XX个数
		{MetricName: "listener_http_code_5xx", AliyunMetricName: "ListenerHTTPCode5XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听后端每秒2XX状态码个数
		{MetricName: "listener_http_code_upstream_2xx", AliyunMetricName: "ListenerHTTPCodeUpstream2XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听后端每秒3XX状态码个数
		{MetricName: "listener_http_code_upstream_3xx", AliyunMetricName: "ListenerHTTPCodeUpstream3XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听后端每秒4XX状态码个数
		{MetricName: "listener_http_code_upstream_4xx", AliyunMetricName: "ListenerHTTPCodeUpstream4XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听后端每秒5XX状态码个数
		{MetricName: "listener_http_code_upstream_5xx", AliyunMetricName: "ListenerHTTPCodeUpstream5XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒自定义Reponse个数
		{MetricName: "listener_http_fixed_response", AliyunMetricName: "ListenerHTTPFixedResponse", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒重定向个数
		{MetricName: "listener_http_redirect", AliyunMetricName: "ListenerHTTPRedirect", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听健康的服务器数
		{MetricName: "listener_healthy_host_count", AliyunMetricName: "ListenerHealthyHostCount", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听入带宽
		{MetricName: "listener_in_bits", AliyunMetricName: "ListenerInBits", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒新建连接数
		{MetricName: "listener_new_connection", AliyunMetricName: "ListenerNewConnection", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒无粘性请求数
		{MetricName: "listener_non_sticky_request", AliyunMetricName: "ListenerNonStickyRequest", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听出带宽
		{MetricName: "listener_out_bits", AliyunMetricName: "ListenerOutBits", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒请求数
		{MetricName: "listener_qps", AliyunMetricName: "ListenerQPS", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听每秒丢弃连接数
		{MetricName: "listener_rejected_connection", AliyunMetricName: "ListenerRejectedConnection", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听不健康的服务器数
		{MetricName: "listener_un_healthy_host_count", AliyunMetricName: "ListenerUnHealthyHostCount", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听后端每秒连接失败数
		{MetricName: "listener_upstream_connection_error", AliyunMetricName: "ListenerUpstreamConnectionError", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听后端应答时间
		{MetricName: "listener_upstream_response_time", AliyunMetricName: "ListenerUpstreamResponseTime", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 监听后端每秒TLS握手失败数
		{MetricName: "listener_upstream_tls_negotiation_error", AliyunMetricName: "ListenerUpstreamTLSNegotiationError", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort"}, Value: true},
		// 负载均衡实例活跃连接数
		{MetricName: "load_balancer_active_connection", AliyunMetricName: "LoadBalancerActiveConnection", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒TLS握手失败连接数
		{MetricName: "load_balancer_client_tls_negotiation_error", AliyunMetricName: "LoadBalancerClientTLSNegotiationError", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒2XX个数
		{MetricName: "load_balancer_http_code_2xx", AliyunMetricName: "LoadBalancerHTTPCode2XX", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒3XX个数
		{MetricName: "load_balancer_http_code_3xx", AliyunMetricName: "LoadBalancerHTTPCode3XX", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒4XX个数
		{MetricName: "load_balancer_http_code_4xx", AliyunMetricName: "LoadBalancerHTTPCode4XX", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒500个数
		{MetricName: "load_balancer_http_code_500", AliyunMetricName: "LoadBalancerHTTPCode500", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒502个数
		{MetricName: "load_balancer_http_code_502", AliyunMetricName: "LoadBalancerHTTPCode502", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒503个数
		{MetricName: "load_balancer_http_code_503", AliyunMetricName: "LoadBalancerHTTPCode503", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒504个数
		{MetricName: "load_balancer_http_code_504", AliyunMetricName: "LoadBalancerHTTPCode504", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒5XX个数
		{MetricName: "load_balancer_http_code_5xx", AliyunMetricName: "LoadBalancerHTTPCode5XX", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例后端每秒2XX个数
		{MetricName: "load_balancer_http_code_upstream_2xx", AliyunMetricName: "LoadBalancerHTTPCodeUpstream2XX", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例后端每秒3XX个数
		{MetricName: "load_balancer_http_code_upstream_3xx", AliyunMetricName: "LoadBalancerHTTPCodeUpstream3XX", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例后端每秒4XX个数
		{MetricName: "load_balancer_http_code_upstream_4xx", AliyunMetricName: "LoadBalancerHTTPCodeUpstream4XX", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例后端每秒5XX个数
		{MetricName: "load_balancer_http_code_upstream_5xx", AliyunMetricName: "LoadBalancerHTTPCodeUpstream5XX", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒自定义Reponse个数
		{MetricName: "load_balancer_http_fixed_response", AliyunMetricName: "LoadBalancerHTTPFixedResponse", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒重定向个数
		{MetricName: "load_balancer_http_redirect", AliyunMetricName: "LoadBalancerHTTPRedirect", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例健康的服务器数
		{MetricName: "load_balancer_healthy_host_count", AliyunMetricName: "LoadBalancerHealthyHostCount", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例入带宽
		{MetricName: "load_balancer_in_bits", AliyunMetricName: "LoadBalancerInBits", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒新建连接数
		{MetricName: "load_balancer_new_connection", AliyunMetricName: "LoadBalancerNewConnection", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒无粘性请求数
		{MetricName: "load_balancer_non_sticky_request", AliyunMetricName: "LoadBalancerNonStickyRequest", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例出带宽
		{MetricName: "load_balancer_out_bits", AliyunMetricName: "LoadBalancerOutBits", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒请求数
		{MetricName: "load_balancer_qps", AliyunMetricName: "LoadBalancerQPS", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例每秒丢弃连接数
		{MetricName: "load_balancer_rejected_connection", AliyunMetricName: "LoadBalancerRejectedConnection", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例不健康的服务器数
		{MetricName: "load_balancer_un_healthy_host_count", AliyunMetricName: "LoadBalancerUnHealthyHostCount", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例后端每秒连接失败数
		{MetricName: "load_balancer_upstream_connection_error", AliyunMetricName: "LoadBalancerUpstreamConnectionError", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例后端应答时间
		{MetricName: "load_balancer_upstream_response_time", AliyunMetricName: "LoadBalancerUpstreamResponseTime", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 负载均衡实例后端每秒TLS握手失败数
		{MetricName: "load_balancer_upstream_tls_negotiation_error", AliyunMetricName: "LoadBalancerUpstreamTLSNegotiationError", Dimensions: []string{"loadBalancerId"}, Value: true},
		// 转发规则后端每秒2XX个数
		{MetricName: "rule_http_code_upstream_2xx", AliyunMetricName: "RuleHTTPCodeUpstream2XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端每秒3XX个数
		{MetricName: "rule_http_code_upstream_3xx", AliyunMetricName: "RuleHTTPCodeUpstream3XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端每秒4XX个数
		{MetricName: "rule_http_code_upstream_4xx", AliyunMetricName: "RuleHTTPCodeUpstream4XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端每秒5XX个数
		{MetricName: "rule_http_code_upstream_5xx", AliyunMetricName: "RuleHTTPCodeUpstream5XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则健康的服务器数
		{MetricName: "rule_healthy_host_count", AliyunMetricName: "RuleHealthyHostCount", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则每秒无粘性请求数
		{MetricName: "rule_non_sticky_request", AliyunMetricName: "RuleNonStickyRequest", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则每秒请求数
		{MetricName: "rule_qps", AliyunMetricName: "RuleQPS", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则不健康的服务器数
		{MetricName: "rule_un_healthy_host_count", AliyunMetricName: "RuleUnHealthyHostCount", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端每秒连接失败数
		{MetricName: "rule_upstream_connection_error", AliyunMetricName: "RuleUpstreamConnectionError", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端应答时间
		{MetricName: "rule_upstream_response_time", AliyunMetricName: "RuleUpstreamResponseTime", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端每秒TLS握手失败数
		{MetricName: "rule_upstream_tls_negotiation_error", AliyunMetricName: "RuleUpstreamTLSNegotiationError", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 服务器组后端每秒2XX个数
		{MetricName: "server_group_http_code_upstream_2xx", AliyunMetricName: "ServerGroupHTTPCodeUpstream2XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组后端每秒3XX个数
		{MetricName: "server_group_http_code_upstream_3xx", AliyunMetricName: "ServerGroupHTTPCodeUpstream3XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组后端每秒4XX个数
		{MetricName: "server_group_http_code_upstream_4xx", AliyunMetricName: "ServerGroupHTTPCodeUpstream4XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组后端每秒5XX个数
		{MetricName: "server_group_http_code_upstream_5xx", AliyunMetricName: "ServerGroupHTTPCodeUpstream5XX", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组健康的服务器数
		{MetricName: "server_group_healthy_host_count", AliyunMetricName: "ServerGroupHealthyHostCount", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组无粘性请求数
		{MetricName: "server_group_non_sticky_request", AliyunMetricName: "ServerGroupNonStickyRequest", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组每秒请求数
		{MetricName: "server_group_qps", AliyunMetricName: "ServerGroupQPS", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组不健康的服务器数
		{MetricName: "server_group_un_healthy_host_count", AliyunMetricName: "ServerGroupUnHealthyHostCount", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组后端每秒连接失败数
		{MetricName: "server_group_upstream_connection_error", AliyunMetricName: "ServerGroupUpstreamConnectionError", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组后端应答时间
		{MetricName: "server_group_upstream_response_time", AliyunMetricName: "ServerGroupUpstreamResponseTime", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 服务器组后端每秒TLS握手失败数
		{MetricName: "server_group_upstream_tls_negotiation_error", AliyunMetricName: "ServerGroupUpstreamTLSNegotiationError", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "serverGroupId"}, Value: true},
		// 转发规则后端每秒2XX占比
		{MetricName: "rule_http_code_upstream_2xx_ratio", AliyunMetricName: "RuleHTTPCodeUpstream2XXRatio", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端每秒3XX占比
		{MetricName: "rule_http_code_upstream_3xx_ratio", AliyunMetricName: "RuleHTTPCodeUpstream3XXRatio", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端每秒4XX占比
		{MetricName: "rule_http_code_upstream_4xx_ratio", AliyunMetricName: "RuleHTTPCodeUpstream4XXRatio", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
		// 转发规则后端每秒5XX占比
		{MetricName: "rule_http_code_upstream_5xx_ratio", AliyunMetricName: "RuleHTTPCodeUpstream5XXRatio", Dimensions: []string{"loadBalancerId", "listenerProtocol", "listenerPort", "ruleId"}, Value: true},
	}
)
