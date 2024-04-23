/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package all

import (
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/cpu"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/disk"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/jvm"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/load"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/mem"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/process"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/swap"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/tcp"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/thread"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/traffic"
)
