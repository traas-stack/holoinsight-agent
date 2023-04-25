/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package mysqlw

import "github.com/traas-stack/holoinsight-agent/pkg/pipeline/integration/base"

type (
	MySQLConf struct {
		base.Conf
		Confs []*MySQLConf_Conf
	}
	MySQLConf_Conf struct {
		Host     string
		Port     int
		User     string
		Password string
	}
)
