package base

import "github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"

type (
	Conf struct {
		Name        string                                 `json:"name"`
		Type        string                                 `json:"type"`
		ExecuteRule *collectconfig.ExecuteRule             `json:"executeRule"`
		RefMetas    map[string]*collectconfig.ElectRegMeta `json:"refMetas"`
	}
)
