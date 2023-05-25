/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package throttle

import (
	"sync/atomic"
	"time"
)

func ThrottleFirst(d time.Duration) func(f func()) {
	wip := int32(0)

	return func(f func()) {
		if atomic.AddInt32(&wip, 1) == 1 {
			go drainLoop(d, &wip, f)
		}
	}
}

func drainLoop(d time.Duration, wip *int32, f func()) {
	delta := atomic.LoadInt32(wip)

	f()

	time.AfterFunc(d, func() {
		if atomic.AddInt32(wip, -delta) != 0 {
			drainLoop(d, wip, f)
		}
	})
}
