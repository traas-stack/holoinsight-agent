/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

const (
	RdsNamespace = "acs_rds_dashboard"
	RdsPrefix    = "alibabacloud_rds"
)

var (
	RdsMetricNames = []*AliyunMetricDesc{
		// 连接数使用率
		{MetricName: "connection_usage", AliyunMetricName: "ConnectionUsage", Average: true, Minimum: true, Maximum: true},
		// CPU使用率
		{MetricName: "cpu_usage", AliyunMetricName: "CpuUsage", Average: true, Minimum: true, Maximum: true},
		// 只读实例延迟
		{MetricName: "data_delay", AliyunMetricName: "DataDelay", Average: true, Minimum: true, Maximum: true},
		// 磁盘使用率
		{MetricName: "disk_usage", AliyunMetricName: "DiskUsage", Average: true, Minimum: true, Maximum: true},
		// IOPS使用率
		{MetricName: "iops_usage", AliyunMetricName: "IOPSUsage", Average: true, Minimum: true, Maximum: true},
		// 内存使用率
		{MetricName: "memory_usage", AliyunMetricName: "MemoryUsage", Average: true, Minimum: true, Maximum: true},
		// MySQL_ActiveSessions
		{MetricName: "mysql_active_sessions", AliyunMetricName: "MySQL_ActiveSessions", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒Delete量
		{MetricName: "mysql_com_delete", AliyunMetricName: "MySQL_ComDelete", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒Insert量
		{MetricName: "mysql_com_insert", AliyunMetricName: "MySQL_ComInsert", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒InsertSelect量
		{MetricName: "mysql_com_insert_select", AliyunMetricName: "MySQL_ComInsertSelect", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒Replace量
		{MetricName: "mysql_com_replace", AliyunMetricName: "MySQL_ComReplace", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒ReplaceSelect量
		{MetricName: "mysql_com_replace_select", AliyunMetricName: "MySQL_ComReplaceSelect", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒Select量
		{MetricName: "mysql_com_select", AliyunMetricName: "MySQL_ComSelect", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒Update量
		{MetricName: "mysql_com_update", AliyunMetricName: "MySQL_ComUpdate", Average: true, Minimum: true, Maximum: true},
		// MySQL_数据磁盘使用量
		{MetricName: "mysql_data_disk_size", AliyunMetricName: "MySQL_DataDiskSize", Average: true, Minimum: true, Maximum: true},
		// MySQL_BP脏页百分率
		{MetricName: "mysql_ibuf_dirty_ratio", AliyunMetricName: "MySQL_IbufDirtyRatio", Average: true, Minimum: true, Maximum: true},
		// MySQL_BP读命中率
		{MetricName: "mysql_ibuf_read_hit", AliyunMetricName: "MySQL_IbufReadHit", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒逻辑读次数
		{MetricName: "mysql_ibuf_request_r", AliyunMetricName: "MySQL_IbufRequestR", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒逻辑写次数
		{MetricName: "mysql_ibuf_request_w", AliyunMetricName: "MySQL_IbufRequestW", Average: true, Minimum: true, Maximum: true},
		// MySQL_BP利用率
		{MetricName: "mysql_ibuf_use_ratio", AliyunMetricName: "MySQL_IbufUseRatio", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒读取数据量
		{MetricName: "mysql_inno_db_data_read", AliyunMetricName: "MySQL_InnoDBDataRead", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒写入数据量
		{MetricName: "mysql_inno_db_data_written", AliyunMetricName: "MySQL_InnoDBDataWritten", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒日志fsync量
		{MetricName: "mysql_inno_db_log_fsync", AliyunMetricName: "MySQL_InnoDBLogFsync", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒日志写请求次数
		{MetricName: "mysql_inno_db_log_write_requests", AliyunMetricName: "MySQL_InnoDBLogWriteRequests", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒日志物理写次数
		{MetricName: "mysql_inno_db_log_writes", AliyunMetricName: "MySQL_InnoDBLogWrites", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒删除行数
		{MetricName: "mysql_inno_db_row_delete", AliyunMetricName: "MySQL_InnoDBRowDelete", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒插入行数
		{MetricName: "mysql_inno_db_row_insert", AliyunMetricName: "MySQL_InnoDBRowInsert", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒读取行数
		{MetricName: "mysql_inno_db_row_read", AliyunMetricName: "MySQL_InnoDBRowRead", Average: true, Minimum: true, Maximum: true},
		// MySQL_InnoDB每秒更新行数
		{MetricName: "mysql_inno_db_row_update", AliyunMetricName: "MySQL_InnoDBRowUpdate", Average: true, Minimum: true, Maximum: true},
		// MySQL_实例磁盘使用量
		{MetricName: "mysql_instance_disk_size", AliyunMetricName: "MySQL_InstanceDiskSize", Average: true, Minimum: true, Maximum: true},
		// MySQL_日志磁盘使用量
		{MetricName: "mysql_log_disk_size", AliyunMetricName: "MySQL_LogDiskSize", Average: true, Minimum: true, Maximum: true},
		// MySQL网络流入带宽
		{MetricName: "mysql_network_in_new", AliyunMetricName: "MySQL_NetworkInNew", Average: true, Minimum: true, Maximum: true},
		// MySQL网络流出带宽
		{MetricName: "mysql_network_out_new", AliyunMetricName: "MySQL_NetworkOutNew", Average: true, Minimum: true, Maximum: true},
		// MySQL_其他磁盘使用量
		{MetricName: "mysql_other_disk_size", AliyunMetricName: "MySQL_OtherDiskSize", Average: true, Minimum: true, Maximum: true},
		// MySQL_ProxyCpu使用率
		{MetricName: "mysql_proxy_cpu_usage", AliyunMetricName: "MySQL_ProxyCpuUsage", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒查询量
		{MetricName: "mysql_qps", AliyunMetricName: "MySQL_QPS", Average: true, Minimum: true, Maximum: true},
		// 只读实例IO线程状态
		{MetricName: "mysql_slave_io_running", AliyunMetricName: "MySQL_SlaveIORunning", Average: true, Minimum: true, Maximum: true},
		// 只读实例SQL线程状态
		{MetricName: "mysql_slave_sql_running", AliyunMetricName: "MySQL_SlaveSQLRunning", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒慢查询量
		{MetricName: "mysql_slow_queries", AliyunMetricName: "MySQL_SlowQueries", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒事务数
		{MetricName: "mysql_tps", AliyunMetricName: "MySQL_TPS", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒创建临时表数量
		{MetricName: "mysql_temp_disk_table_creates", AliyunMetricName: "MySQL_TempDiskTableCreates", Average: true, Minimum: true, Maximum: true},
		// MySQL_线程连接数
		{MetricName: "mysql_threads_connected", AliyunMetricName: "MySQL_ThreadsConnected", Average: true, Minimum: true, Maximum: true},
		// MySQL_活跃线程数
		{MetricName: "mysql_threads_running", AliyunMetricName: "MySQL_ThreadsRunning", Average: true, Minimum: true, Maximum: true},
		// MySQL_临时磁盘使用量
		{MetricName: "mysql_tmp_disk_size", AliyunMetricName: "MySQL_TmpDiskSize", Average: true, Minimum: true, Maximum: true},
		// MySQL每秒物理读次数
		{MetricName: "mysqlibuf_pool_reads", AliyunMetricName: "MySQL_ibufPoolReads", Average: true, Minimum: true, Maximum: true},
		// SQLServer网络流入带宽
		{MetricName: "sqlserver_network_in_new", AliyunMetricName: "SQLServer_NetworkInNew", Average: true, Minimum: true, Maximum: true},
		// SQLServer网络流出带宽
		{MetricName: "sqlserver_network_out_new", AliyunMetricName: "SQLServer_NetworkOutNew", Average: true, Minimum: true, Maximum: true},
	}
)
