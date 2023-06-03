/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"encoding/gob"
	"errors"
	"time"
)

type (
	// AlignedTimer emit aligned time.Time. [align+offset, 2*align+offset, 3*align+offset, ...]
	// When skipExpiredWindow is true, if wall time is after next 'n*align+offset' when calling Next(),
	// it is skipped, and the next emit time will be (n+1)*align+offset.
	// When skipExpiredWindow is false, if wall time is after next 'n*align+offset' when calling Next(),
	// it will trigger an emit right now.
	// If skipFirstTimer is true, it ensures that the (first emit time - now) >= align.
	// Otherwise, first emit may be trigger within 'align' time.
	// Example: align=5s offset=2s, skipExpiredWindow=true, skipFirstTimer=false
	// Now is 00:00:6s, the emitted times will be: 00:00:07, 00:00:12, 00:00:17, ...
	// Assuming task executing at 00:00:07 is very heavy, and when it is completed, the time exceeds 00:00:12. So the next emit time is 00:00:17.
	// The emitted times will be: 00:00:07, 00:00:17, ...
	AlignedTimer struct {
		timer             *time.Timer
		skipExpiredWindow bool
		align             time.Duration
		offset            time.Duration
		nextEmitTime      time.Time
		// C returns the time chan.
		// Note that the value in chan may be different with the value returned by Next().
		// Please refer to the value returned by Next().
		C <-chan time.Time
	}
	alignedTimerStateObj struct {
		SkipExpiredWindow bool
		Align             time.Duration
		Offset            time.Duration
		NextEmitTime      time.Time
	}
)

func init() {
	gob.Register(&alignedTimerStateObj{})
}

func NewAlignedTimer(align, offset time.Duration, skipExpiredWindow bool, skipFirstTimer bool) (*AlignedTimer, time.Time) {
	var timer *time.Timer
	now := time.Now()

	var nextEmitTime time.Time
	if now.Sub(now.Truncate(align).Add(offset)) > 0 || skipFirstTimer {
		nextEmitTime = now.Truncate(align).Add(align).Add(offset)
		timer = time.NewTimer(nextEmitTime.Sub(now))
	} else {
		nextEmitTime = now.Truncate(align).Add(offset)
		timer = time.NewTimer(nextEmitTime.Sub(now))
	}

	return &AlignedTimer{
		timer:             timer,
		align:             align,
		offset:            offset,
		skipExpiredWindow: skipExpiredWindow,
		nextEmitTime:      nextEmitTime,
		C:                 timer.C,
	}, nextEmitTime
}

// Next triggers emit, returns next emit time.
// The returned time may before time.Now() if 'skipExpiredWindow' is false.
func (t *AlignedTimer) Next() time.Time {
	now := time.Now()

	if t.skipExpiredWindow {
		if now.Sub(now.Truncate(t.align).Add(t.offset)) > 0 {
			t.nextEmitTime = now.Truncate(t.align).Add(t.align).Add(t.offset)
		} else {
			t.nextEmitTime = now.Truncate(t.align).Add(t.offset)
		}
	} else {
		t.nextEmitTime = t.nextEmitTime.Add(t.align)
	}

	d := t.nextEmitTime.Sub(now)
	if d <= 0 {
		t.timer.Reset(0)
	} else {
		t.timer.Reset(t.nextEmitTime.Sub(now))
	}

	return t.nextEmitTime
}

// Stop tops the timer
func (t *AlignedTimer) Stop() {
	t.timer.Stop()
}

// NextEmitTime returns expected next emit time.
// It may have: 'next emit time' < 'time.Now()'
func (t *AlignedTimer) NextEmitTime() time.Time {
	return t.nextEmitTime
}

// SaveState saves current timer state into []byte
func (t *AlignedTimer) SaveState() ([]byte, error) {
	state := &alignedTimerStateObj{
		SkipExpiredWindow: t.skipExpiredWindow,
		Align:             t.align,
		Offset:            t.offset,
		NextEmitTime:      t.nextEmitTime,
	}
	return GobEncode(state)
}

// LoadState loads timer state from []byte
func (t *AlignedTimer) LoadState(b []byte) error {
	state := &alignedTimerStateObj{}
	if err := GobDecode(b, state); err != nil {
		return err
	}

	if state.Align == t.align && state.Offset == t.offset && state.SkipExpiredWindow == t.skipExpiredWindow {
		t.timer.Stop()

		delay := state.NextEmitTime.Sub(time.Now())
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)

		t.nextEmitTime = state.NextEmitTime
		t.timer = timer
		t.C = timer.C
		return nil
	}

	return errors.New("timer state changed")
}
