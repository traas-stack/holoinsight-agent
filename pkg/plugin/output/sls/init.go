/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package sls

import "github.com/traas-stack/holoinsight-agent/pkg/plugin/output"

func init() {
	output.Register("sls", func(config output.Config) (output.Output, error) {
		return NewSLSOutput()
	})
}
