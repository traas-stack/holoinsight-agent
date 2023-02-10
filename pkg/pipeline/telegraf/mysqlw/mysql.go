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
