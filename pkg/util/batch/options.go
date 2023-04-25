/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package batch

import "time"

type (
	options struct {
		interval       time.Duration
		batchThreshold int

		calculator func(interface{}) int
		weightMax  int
	}

	OptionFunc func(*options)
)

var (
	defaultOptions = options{
		interval: time.Second * 10,
	}
)

func WithMaxWaitStrategy(i time.Duration) OptionFunc {
	return func(o *options) {
		o.interval = i
	}
}

func WithBatchThresholdStrategy(n int) OptionFunc {
	return func(o *options) {
		o.batchThreshold = n
	}
}

func WithItemsWeightStrategy(calculator func(i interface{}) int, threshold int) OptionFunc {
	return func(o *options) {
		o.calculator = calculator
		o.weightMax = threshold
	}
}
