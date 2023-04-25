/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package gateway

import (
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
)

func init() {
	output.Register("cmgateway", newGatewayOutput)
	output.Register("gateway", newGatewayOutput)
}
