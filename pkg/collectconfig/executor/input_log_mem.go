/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

type (
	MemLogInput struct {
		lines []string
	}
)

func NewMemLogInput(lines []string) *MemLogInput {
	return &MemLogInput{
		lines: lines,
	}
}

func (m *MemLogInput) Finished() bool {
	return m.lines == nil
}

func (m *MemLogInput) Start() {
}

func (m *MemLogInput) Stop() {
}

func (m *MemLogInput) Pull(request *PullRequest) (*PullResponse, error) {
	lines := m.lines
	// m.lines = nil
	return &PullResponse{
		Lines:       lines,
		Continued:   true,
		HasMore:     false,
		HasBroken:   false,
		HasBuffer:   false,
		BeginOffset: 0,
		EndOffset:   0,
		FileLength:  0,
		FileId:      "",
	}, nil
}
