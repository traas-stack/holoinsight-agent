/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import "sync"

type (
	StopSignal struct {
		mutex    sync.Mutex
		C        chan struct{}
		stoppedC chan struct{}
	}
)

func NewStopSignal() *StopSignal {
	return &StopSignal{
		C:        make(chan struct{}),
		stoppedC: make(chan struct{}),
	}
}

func (s *StopSignal) StopAndWait() {
	s.Stop()
	s.WaitStopped()
}

// Stop must only be call once
func (s *StopSignal) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.IsStopAsked() {
		close(s.C)
	}
}

func (s *StopSignal) IsStopAsked() bool {
	select {
	case <-s.C:
		return true
	default:
		return false
	}
}

// StopDone must only be call once
func (s *StopSignal) StopDone() {
	close(s.stoppedC)
}

func (s *StopSignal) WaitStopped() {
	<-s.stoppedC
}
