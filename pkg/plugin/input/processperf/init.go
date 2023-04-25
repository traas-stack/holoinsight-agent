/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package processperf

import "github.com/traas-stack/holoinsight-agent/pkg/plugin/input"

func init() {
	input.Register("processperf", func(config input.Config) (input.Input, error) {
		return newProcessPerfInput(config)
	})
}
