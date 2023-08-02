/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"context"
	"io"
	"net"
	"time"
)

type (
	// ReaderCloserFunc wraps an io.Reader. It calls Closer func when reader meets io.EOF.
	ReaderCloserFunc struct {
		Reader io.Reader
		Closer func()
	}
	ReaderReadHook struct {
		Reader io.Reader
		Before func()
		After  func(p []byte, n int, err error)
	}
	WriterWriteHook struct {
		Writer io.Writer
		Before func()
		After  func(p []byte, n int, err error)
	}
	ReadWriterConn struct {
		io.Reader
		io.Writer
		Cancel context.CancelFunc
	}
)

func (r *ReadWriterConn) Close() error {
	MaybeIOClose(r.Reader)
	MaybeIOClose(r.Writer)
	if r.Cancel != nil {
		r.Cancel()
	}
	return nil
}

func (r *ReadWriterConn) LocalAddr() net.Addr {
	return nil
}

func (r *ReadWriterConn) RemoteAddr() net.Addr {
	return nil
}

func (r *ReadWriterConn) SetDeadline(t time.Time) error {
	return nil
}

func (r *ReadWriterConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (r *ReadWriterConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (r *ReadWriterConn) CloseRead() error {
	return MaybeIOClose(r.Reader)
}

func (r *ReadWriterConn) CloseWrite() error {
	return MaybeIOClose(r.Writer)
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

func (s *ReaderReadHook) Read(p []byte) (int, error) {
	if s.Before != nil {
		s.Before()
	}
	n, err := s.Reader.Read(p)
	if s.After != nil {
		s.After(p, n, err)
	}
	return n, err
}

func (s *ReaderReadHook) Close() error {
	if x, ok := s.Reader.(io.Closer); ok {
		return x.Close()
	}
	return nil
}

func (s *WriterWriteHook) Write(p []byte) (int, error) {
	if s.Before != nil {
		s.Before()
	}
	n, err := s.Writer.Write(p)
	if s.After != nil {
		s.After(p, n, err)
	}
	return n, err
}

func (s *WriterWriteHook) Close() error {
	if x, ok := s.Writer.(io.Closer); ok {
		return x.Close()
	}
	return nil
}

func MaybeIOClose(i interface{}) error {
	if x, ok := i.(io.Closer); ok {
		return x.Close()
	}
	return nil
}

type setDeadline interface {
	SetDeadline(t time.Time) error
}

func MaybeSetDeadline(i interface{}, d time.Time) error {
	if x, ok := i.(setDeadline); ok {
		return x.SetDeadline(d)
	}
	return nil
}

func CopyConn(ctx context.Context, conn1 io.ReadWriter, conn2 io.ReadWriter, timeout time.Duration) error {
	errCh := make(chan error, 2)

	// write output to conn
	go func() {
		_, err := io.Copy(conn1, &ReaderReadHook{Reader: conn2, Before: func() {
			MaybeSetDeadline(conn2, time.Now().Add(timeout))
		}})
		errCh <- err
		MaybeCloseWrite(conn1)
		MaybeCloseRead(conn2)
	}()

	// read input from conn
	go func() {
		_, err := io.Copy(conn2, &ReaderReadHook{Reader: conn1, Before: func() {
			MaybeSetDeadline(conn1, time.Now().Add(timeout))
		}})
		errCh <- err
		MaybeCloseWrite(conn2)
		MaybeCloseRead(conn1)
	}()

	wait := 2
	for {
		select {
		case err := <-errCh:
			if err != nil && err != io.EOF {
				return err
			}
			wait--
			if wait == 0 {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
