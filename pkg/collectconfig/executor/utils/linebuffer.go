/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package utils

import (
	"bytes"
)

const (
	line_separator_r = '\r'
	line_separator_n = '\n'
)

type (
	// 积累bytes的缓冲器, 可以扫描出多个行
	LineBuffer struct {
		buffer  []byte
		add     []byte
		maxSize int
		// 如果遇到超长行就进入损坏状态
		broken bool
	}
	LineBufferStateObj struct {
		Buffer []byte
		Add    []byte
		Broken bool
	}
)

func NewLineBuffer(maxSize int) *LineBuffer {
	return &LineBuffer{
		maxSize: maxSize,
	}
}

// 往buffer里添加一些bytes
func (buf *LineBuffer) Add(a []byte) {
	if buf.broken {
		// 处于损坏状态, 则跳过这个超长行
		// a里必须要包含\n, 否则根本没有处理的必要
		index := bytes.IndexByte(a, line_separator_n)
		if index < 0 {
			return
		}
		a = a[index+1:]
		// 解除损坏状态
		buf.buffer = nil
		buf.broken = false
	}
	buf.add = a
}

// for test only
func (buf *LineBuffer) addString(str string) {
	buf.Add([]byte(str))
}

func (buf *LineBuffer) checkNewLen(newLen int) bool {
	if newLen <= buf.maxSize {
		return true
	}
	buf.buffer = nil
	buf.add = nil
	buf.broken = true
	return false
}

func (buf *LineBuffer) IsBroken() bool {
	return buf.broken
}

// 尝试解析出一行, 如果成功则返回 line, true 否则返回 "", false
func (buf *LineBuffer) Next() ([]byte, bool) {
	// 如果buffer有残留, 那么可能一行被切成两半, 开头在buffer里, 尾巴在add里
	if len(buf.buffer) > 0 {
		// 在尾巴里找\n
		index := bytes.IndexByte(buf.add, line_separator_n)
		// 没找到, 将add追加入buffer
		if index < 0 {
			// 大小防御
			if !buf.checkNewLen(len(buf.buffer) + len(buf.add)) {
				return nil, false
			}
			buf.buffer = append(buf.buffer, buf.add...)
			buf.add = nil
			return nil, false
		}

		// 找到了\n
		bufferLen := len(buf.buffer)
		temp := make([]byte, bufferLen+index)
		// temp[0, bufferLen+index) = buffer[0, bufferLen) + add[0, index)
		copy(temp[0:bufferLen], buf.buffer[0:bufferLen])
		copy(temp[bufferLen:], buf.add[0:index])

		buf.buffer = nil
		buf.add = buf.add[index+1:]

		return trim(temp), true
	}

	// buffer 已经消费完了, 从add里找下一个行
	index := bytes.IndexByte(buf.add, line_separator_n)
	// 没找到
	if index < 0 {
		// 大小防御
		if !buf.checkNewLen(len(buf.buffer) + len(buf.add)) {
			return nil, false
		}
		buf.buffer = append(buf.buffer, buf.add...)
		buf.add = nil
		return nil, false
	}

	// 解码动作交给外部去做
	line := buf.add[0:index]
	buf.add = buf.add[index+1:]
	return trim(line), true
}

func (buf *LineBuffer) Clear() {
	buf.buffer = nil
	buf.add = nil
	buf.broken = false
}

// 数一下有多少行是可用的, 就是数一下\n的个数
func (buf *LineBuffer) AvailableLines() int {
	return countLine(buf.buffer) + countLine(buf.add)
}

func (buf *LineBuffer) Empty() bool {
	return len(buf.buffer) == 0 && len(buf.add) == 0
}
func (buf *LineBuffer) SaveState() *LineBufferStateObj {
	return &LineBufferStateObj{
		Buffer: buf.buffer,
		Add:    buf.add,
		Broken: buf.broken,
	}
}
func (buf *LineBuffer) LoadState(state *LineBufferStateObj) {
	buf.add = state.Add
	buf.buffer = state.Buffer
	buf.broken = state.Broken
}

func countLine(bs []byte) int {
	count := 0
	for len(bs) > 0 {
		index := bytes.IndexByte(bs, line_separator_n)
		if index < 0 {
			break
		}
		count += 1
		bs = bs[index+1:]
	}
	return count
}

// trim \r
func trim(b []byte) []byte {
	blen := len(b)
	if blen > 0 && b[blen-1] == line_separator_r {
		return b[0 : blen-1]
	}
	return b
}
