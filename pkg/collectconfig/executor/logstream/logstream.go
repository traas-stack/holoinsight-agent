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

type (
	LogStream interface {
		Start()
		Stop()
		Read(*ReadRequest) (*ReadResponse, int64, error)
		AddListener(Listener) int64
		RemoveListener(Listener, int64)
		Stat() Stat
		Clean()
	}
	Stat struct {
		LatestCursor int64
		PendingBytes int64
		PendingReads int32
	}
	Listener interface {
		Changed(string, LogStream, int64)
	}
	ReadRequest struct {
		Cursor int64
	}
	ReadResponse struct {
		// current read cursor
		Cursor int64
		// next read cursor
		NextCursor int64
		// io(read) start time
		IOStartTime time.Time
		// error when read (such as 'no such file or directory')
		Error error

		// 这里不会合并多行, 也不会解析时间戳, 需要上层自己去处理
		Lines []string `json:"-"`

		// 数据是否是连续的
		// 我们规定第一个PullResult总是连续的
		Continued bool
		// whether file has more available data to read
		HasMore bool

		// 如果为true表示已经日志流已经结束
		Finished bool

		// 一些内部信息, 可以依赖他们去调试
		HasBroken  bool
		HasBuffer  bool
		FileLength int64
		FileId     string
		Path       string

		BeginOffset int64
		EndOffset   int64

		// 剩余可读次数
		remainCount int32

		decodeMutex  sync.Mutex
		decodedCache map[string][]string
	}
)

func (resp *ReadResponse) Bytes() int64 {
	return resp.EndOffset - resp.BeginOffset
}

// GetDecodedLines returns lines decoded using specified charset
func (resp *ReadResponse) GetDecodedLines(charset string) ([]string, error) {
	if charset == "" || charset == text.UTF8 {
		// There is a situation: configure charset=UTF-8, but the actual data is gb18030. Garbled characters will still be generated at this time.
		return resp.Lines, nil
	}

	// TODO AUTO detect from data ?

	supportedEncoding := text.GetEncoding(charset)
	if supportedEncoding == nil {
		return nil, errors.New("unsupported charset: " + charset)
	}

	resp.decodeMutex.Lock()
	defer resp.decodeMutex.Unlock()

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
