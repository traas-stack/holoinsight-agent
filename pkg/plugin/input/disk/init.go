/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package disk

import (
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
)

func init() {
	input.Register("disk", func(config input.Config) (api.Input, error) {
		return &diskInput{}, nil
	})
}
