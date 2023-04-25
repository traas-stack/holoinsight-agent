/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package accumulator

import "github.com/traas-stack/holoinsight-agent/pkg/model"

type (
	// Like Telegraf.Accumulator
	Accumulator interface {
		AddMetric(*model.Metric)
		AddError(error)
	}
)
