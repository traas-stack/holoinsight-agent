/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import "io"

// ReaderCloserFunc wraps an io.Reader. It calls Closer func when reader meets io.EOF.
type ReaderCloserFunc struct {
	Reader io.Reader
	Closer func()
}

func (s *ReaderCloserFunc) Read(p []byte) (int, error) {
	n, err := s.Reader.Read(p)
	if err == io.EOF {
		if s.Closer != nil {
			s.Closer()
		}
	}
	return n, err
}
