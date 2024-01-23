/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

type (
	logStreamReaderImpl struct {
		key      string
		m        *Manager
		ls       LogStream
		cursor   int64
		listener Listener
	}
)

func (l *logStreamReaderImpl) Cursor() int64 {
	return l.cursor
}

func (l *logStreamReaderImpl) LoadReadState(cursor int64) error {
	if err := l.ls.LoadReadState(cursor); err != nil {
		return err
	}

	l.cursor = cursor
	return nil
}

func (l *logStreamReaderImpl) Read() (*ReadResponse, error) {
	resp, nextCursor, err := l.ls.Read(l.cursor)
	l.cursor = nextCursor
	return resp, err
}

func (l *logStreamReaderImpl) SetListener(listener Listener) {
	l.listener = listener
}

func (l *logStreamReaderImpl) Release() {
	l.m.Release(l.ls)
}
