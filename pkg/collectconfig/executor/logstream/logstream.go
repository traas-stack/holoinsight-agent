/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import "time"

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
	}
)

func (resp *ReadResponse) Bytes() int64 {
	return resp.EndOffset - resp.BeginOffset
}
