/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/text"
	"golang.org/x/text/encoding"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	ModeLine     Mode = "line"
	ModeLogGroup Mode = "loggroup"
)

type (
	Mode      string
	LogStream interface {
		// GetKey returns the key of LogStream
		GetKey() string
		Start()
		Stop()
		Read(int64) (*ReadResponse, int64, error)
		AddListener(Listener) int64
		RemoveListener(Listener, int64)
		Stat() Stat
		Clean()
		LoadReadState(cursor int64) error
	}
	LogStreamState interface {
		SaveState() (interface{}, error)
		LoadState(interface{}) error
	}
	Reader interface {
		// Returns current cursor
		Cursor() int64
		LoadReadState(int64) error
		// Read
		Read() (*ReadResponse, error)
		// Set listener
		SetListener(Listener)
		Release()
	}
	Stat struct {
		LatestCursor int64
		PendingBytes int64
		PendingReads int32
	}
	Listener interface {
		Changed(LogStream, int64)
	}
	ReadRequest struct {
		Cursor int64
	}
	// ReadResponse. The caller must not modify the structure.
	ReadResponse struct {
		// current read cursor
		Cursor int64
		// io read start time
		IOStartTime time.Time
		// io read end time
		IOEndTime time.Time
		// error when read (such as 'no such file or directory')
		error error

		Lines     []string `json:"-"`
		LogGroups []*LogGroup

		// Whether the next data can be read immediately
		HasMore   bool
		HasBroken bool

		Path string
		// The number of bytes read.
		// For some implementations, this value may be inaccurate.
		Bytes int64
		Count int
		// Use a string to describe the scope of this read.
		Range string
		// Count of \u0000 of this read
		ZeroBytes int

		decodeMutex  sync.Mutex
		decodedCache map[string][]string
	}
	cachedRead struct {
		pendingReads int32
		resp         *ReadResponse
	}
	LogGroup struct {
		Tags map[string]string
		Logs []*Log
	}
	Log struct {
		Time     int64
		Contents map[string]string
	}
)

func (resp *ReadResponse) IsEmpty() bool {
	return len(resp.Lines) == 0 && len(resp.LogGroups) == 0
}

func (resp *ReadResponse) IOCost() time.Duration {
	return resp.IOEndTime.Sub(resp.IOStartTime)
}

// GetDecodedLines returns lines decoded using specified charset
func (resp *ReadResponse) GetDecodedLines(charset string) ([]string, error) {
	if charset == "" || charset == text.UTF8 {
		// There is a situation: configure charset=UTF-8, but the actual data is gb18030. Garbled characters will still be generated at this time.
		return resp.Lines, nil
	}

	supportedEncoding := text.GetEncoding(charset)
	if supportedEncoding == nil {
		return nil, errors.New("unsupported charset: " + charset)
	}

	resp.decodeMutex.Lock()
	defer resp.decodeMutex.Unlock()

	// lazy init
	if resp.decodedCache == nil {
		resp.decodedCache = make(map[string][]string)
	}

	if cached, ok := resp.decodedCache[charset]; ok {
		return cached, nil
	}

	decoded := make([]string, len(resp.Lines))
	decoder := supportedEncoding.NewDecoder()

	for i, line := range resp.Lines {
		if d, err := maybeDecode(line, decoder); err == nil {
			decoded[i] = d
		} else {
			decoded[i] = line
		}
	}

	resp.decodedCache[charset] = decoded
	return decoded, nil
}

// maybeDecode decodes string to utf8
// If the given string is already a valid utf8 string, returns itself.
func maybeDecode(s string, decoder *encoding.Decoder) (string, error) {
	if utf8.ValidString(s) {
		return s, nil
	}
	return decoder.String(s)
}
