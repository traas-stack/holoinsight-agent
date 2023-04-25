/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package telegraf

import "github.com/influxdata/telegraf"

type (
	InputWrapper struct {
		input telegraf.Input
	}
)
