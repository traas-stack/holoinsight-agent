package fordev

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway"
	"strings"
)

func NewDevGateway() (*gateway.Service, error) {
	g := appconfig.StdAgentConfig.Gateway
	addr := g.Addr
	if !strings.Contains(addr, ":") {
		addr += ":19610"
	}
	return gateway.New(gateway.Config{
		Addr:         addr,
		Secure:       g.Secure,
		Apikey:       appconfig.StdAgentConfig.ApiKey,
		ServerName:   g.ServiceName,
		CaCertBase64: g.CaCert,
	})
}
