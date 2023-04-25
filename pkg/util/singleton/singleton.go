/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package singleton

import (
	"sync"
)

type (
	Holder struct {
		new      func() (interface{}, error)
		destroy  func(interface{})
		refCount int
		mutex    sync.Mutex
		instance interface{}
		err      error
	}
)

func NewHolder(new func() (interface{}, error), destroy func(interface{})) *Holder {
	return &Holder{
		new:     new,
		destroy: destroy,
	}
}

func (s *Holder) Acquire() (interface{}, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.err != nil {
		return nil, s.err
	}

	if s.instance == nil {
		s.instance, s.err = s.new()
	}

	if s.instance != nil {
		s.refCount++
	}

	return s.instance, s.err
}

func (s *Holder) Release() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.err != nil {
		return
	}

	s.refCount--

	if s.refCount == 0 {
		s.destroy(s.instance)
		s.instance = nil
		s.err = nil
	}
}
