/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package console

import "github.com/traas-stack/holoinsight-agent/pkg/plugin/output"

func init() {
	output.Register("console", func(config output.Config) (output.Output, error) {
		return NewConsoleOutput()
	})
}
