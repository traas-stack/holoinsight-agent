package logstream

import (
	"time"
)

type (
	MemLogStream struct {
		lines []string
	}
)

func NewMemLogStream(lines []string) *MemLogStream {
	return &MemLogStream{
		lines: lines,
	}
}

func (m *MemLogStream) Start() {
}

func (m *MemLogStream) Stop() {
}

func (m *MemLogStream) Read(*ReadRequest) (*ReadResponse, int64, error) {
	return &ReadResponse{
		IOStartTime: time.Now(),
		Lines:       m.lines,
		Continued:   true,
		HasMore:     false,
		Finished:    true,
	}, 0, nil
}

func (m *MemLogStream) AddListener(listener Listener) int64 {
	return 0
}

func (m *MemLogStream) RemoveListener(listener Listener, i int64) {
	// noop
}

func (m *MemLogStream) Stat() Stat {
	return Stat{}
}

func (m *MemLogStream) Clean() {
}
