/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package transfer

import "errors"

type (
	MemoryStateStore struct {
		State map[string]interface{}
	}
	sub struct {
		inner  *MemoryStateStore
		prefix string
	}
)

func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		State: make(map[string]interface{}),
	}
}

func (m *MemoryStateStore) Get(key string) (interface{}, error) {
	if b, ok := m.State[key]; ok {
		return b, nil
	}
	return nil, errors.New("no state for key: " + key)
}

func (m *MemoryStateStore) Put(key string, state interface{}) {
	m.State[key] = state
}

func (m *MemoryStateStore) Sub(prefix string) StateStore {
	return &sub{
		inner:  m,
		prefix: prefix,
	}
}

func (s *sub) Get(key string) (interface{}, error) {
	return s.inner.Get(s.prefix + "@" + key)
}

func (s *sub) Put(key string, state interface{}) {
	s.inner.Put(s.prefix+"@"+key, state)
}

func (s *sub) Sub(prefix string) StateStore {
	return &sub{
		prefix: s.prefix + "@" + prefix,
		inner:  s.inner,
	}
}
