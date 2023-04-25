/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

type (
	AliyunMetricDesc struct {
		MetricName       string
		AliyunMetricName string
		Dimensions       []string
		Average          bool
		Minimum          bool
		Maximum          bool
		Value            bool
		Sum              bool
		Max              bool
	}
)
