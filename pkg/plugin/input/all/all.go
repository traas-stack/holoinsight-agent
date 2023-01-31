package all

import (
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/cpu"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/disk"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/jvm"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/load"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/mem"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/process"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/processperf"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/swap"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/tcp"
	_ "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input/traffic"
)
