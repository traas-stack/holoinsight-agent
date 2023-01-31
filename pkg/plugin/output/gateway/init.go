package gateway

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/plugin/output"
)

func init() {
	output.Register("cmgateway", newGatewayOutput)
	output.Register("gateway", newGatewayOutput)
}
