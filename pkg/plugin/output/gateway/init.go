package gateway

import (
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
)

func init() {
	output.Register("cmgateway", newGatewayOutput)
	output.Register("gateway", newGatewayOutput)
}
