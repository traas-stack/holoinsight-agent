/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package base

import "github.com/traas-stack/holoinsight-agent/pkg/collectconfig"

type (
	Conf struct {
		Name        string                                 `json:"name"`
		Type        string                                 `json:"type"`
		ExecuteRule *collectconfig.ExecuteRule             `json:"executeRule"`
		RefMetas    map[string]*collectconfig.ElectRegMeta `json:"refMetas"`
	}
)
