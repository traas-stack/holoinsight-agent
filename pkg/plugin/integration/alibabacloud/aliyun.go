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
