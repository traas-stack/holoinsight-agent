/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package trigger

import "time"

type (
	TriggerContext interface{}
	Trigger        interface {
		Next(TriggerContext) time.Time
	}
	fixedDelay struct {
		interval time.Duration
	}
	fixedRate struct {
		align  time.Duration
		offset time.Duration
	}
)

func WithFixedDelay(interval time.Duration) Trigger {
	return &fixedDelay{
		interval: interval,
	}
}

func WithFixedRate(align, offset time.Duration) Trigger {
	return &fixedRate{
		align:  align,
		offset: offset,
	}
}

func (f *fixedDelay) Next(context TriggerContext) time.Time {
	return time.Now().Add(f.interval)
}

func (f *fixedRate) Next(context TriggerContext) time.Time {
	return time.Now().Truncate(f.align).Add(f.align).Add(f.offset)
}
