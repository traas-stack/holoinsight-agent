/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/utils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const (
	expireTimeout             = 3 * time.Minute
	DiscardLineWithZeroBytes  = true
	DiscardZeroBytesThreshold = 4096
)

type (
	FileConfig struct {
		Path           string
		MaxLineSize    int
		MaxIOReadBytes int64
	}
	fileSubLogStream struct {
		g      *GLogStream
		config FileConfig
		fileState
	}
	fileState struct {
		file            *os.File
		filestat        os.FileInfo
		offset          int64
		lineBuffer      *utils.LineBuffer
		ignoreFirstLine bool
		inode           uint64
		fileChanged     bool
	}
	fileStateObj struct {
		Cursor int64
		Cache  map[int64]*ReadResponse

		Offset          int64
		LineBuffer      *utils.LineBufferStateObj
		IgnoreFirstLine bool
		Inode           uint64
		FileChanged     bool
	}
)

var (
	DefaultFileConfig = FileConfig{
		MaxLineSize:    1 * 1024 * 1024,
		MaxIOReadBytes: 4 * 1024 * 1024,
	}
	TruncatedErr   = errors.New("truncated")
	FileChangedErr = errors.New("file changed")
)

func init() {
	gob.Register(&fileStateObj{})
}

func NewFileLogStream(config FileConfig) *GLogStream {
	if config.MaxIOReadBytes < DefaultFileConfig.MaxIOReadBytes {
		config.MaxIOReadBytes = DefaultFileConfig.MaxIOReadBytes
	}
	if config.MaxLineSize < DefaultFileConfig.MaxLineSize {
		config.MaxLineSize = DefaultFileConfig.MaxLineSize
	}

	g := &GLogStream{Key: config.Path}
	sub := &fileSubLogStream{
		g:      g,
		config: config,
		fileState: fileState{
			lineBuffer: utils.NewLineBuffer(config.MaxLineSize),
		},
	}
	g.sub = sub
	return g
}

func (f *fileSubLogStream) CreateResponse(cursor int64) *ReadResponse {
	return &ReadResponse{
		Cursor:      cursor,
		IOStartTime: time.Now(),
		Path:        f.config.Path,
	}
}

func (f *fileSubLogStream) Mode() Mode {
	return ModeLine
}

func (f *fileSubLogStream) Start() {
}

func (f *fileSubLogStream) Stop() {
	f.g.Mutex.Lock()
	defer f.g.Mutex.Unlock()

	f.closeFile()
	// TODO need clean?
}

func (f *fileSubLogStream) LoadState(i interface{}) error {
	if i == nil {
		return nil
	}

	state := i.(*fileStateObj)

	if err := f.ensureOpened(true); err != nil {
		return err
	}

	if f.inode != state.Inode {
		f.closeFile()
		return FileChangedErr
	}

	if state.Offset > f.filestat.Size() {
		f.closeFile()
		return TruncatedErr
	}

	f.offset = state.Offset
	f.g.Cursor = state.Cursor
	f.lineBuffer.LoadState(state.LineBuffer)
	f.ignoreFirstLine = state.IgnoreFirstLine
	f.fileChanged = state.FileChanged

	for key, resp := range state.Cache {
		f.g.Cache.Store(key, &cachedRead{
			pendingReads: 0,
			resp:         resp,
		})
		f.g.UpdatePending(resp, true)
	}

	return nil
}

func (f *fileSubLogStream) SaveState() (interface{}, error) {
	if f.file == nil {
		return nil, nil
	}

	cache := make(map[int64]*ReadResponse)
	f.g.Cache.Range(func(key, value any) bool {
		cache[key.(int64)] = value.(*cachedRead).resp
		return true
	})

	return &fileStateObj{
		Offset:          f.offset,
		Cache:           cache,
		Cursor:          f.g.Cursor,
		LineBuffer:      f.lineBuffer.SaveState(),
		IgnoreFirstLine: f.ignoreFirstLine,
		Inode:           f.inode,
		FileChanged:     f.fileChanged,
	}, nil
}

func (f *fileSubLogStream) getFromCache(cursor int64) *ReadResponse {
	if c, ok := f.g.Cache.Load(cursor); ok {
		cached := c.(*cachedRead)
		resp := cached.resp

		if atomic.AddInt32(&cached.pendingReads, -1) == 0 {
			f.g.Cache.Delete(cursor)
			f.g.UpdatePending(resp, false)
		}

		if !resp.HasMore {
			fc := atomic.LoadInt64(&f.g.Cursor)
			if resp.Cursor+1 < fc {
				resp.HasMore = true
			}
		}

		return resp
	}
	return nil
}

func (f *fileSubLogStream) Read(resp *ReadResponse) error {
	if err := f.ensureOpened(true); err != nil {
		return err
	}

	if !f.fileChanged {
		if nowstat, err := os.Stat(f.config.Path); err != nil {
			f.fileChanged = true
		} else if os.SameFile(nowstat, f.filestat) {
			f.filestat = nowstat
		} else {
			f.fileChanged = true
			// get stat by fd
			if st, err := f.file.Stat(); err == nil {
				f.filestat = st
			}
		}
	}

	fileLength := f.filestat.Size()

	if fileLength < f.offset {
		// truncated
		f.closeFile()
		resp.HasMore = true
		f.ensureOpened(false)
		return TruncatedErr
	}

	beginOffset := f.offset

	if f.offset < fileLength {
		readBytes := fileLength - f.offset
		if readBytes > f.config.MaxIOReadBytes {
			readBytes = f.config.MaxIOReadBytes
		}
		buf := make([]byte, readBytes)
		n, err := f.file.ReadAt(buf, f.offset)
		resp.IOEndTime = time.Now()
		buf = buf[:n]
		resp.ZeroBytes = bytes.Count(buf, []byte{0})
		if err != nil && err != io.EOF {
			f.closeFile()
			logger.Errorz("[logstream] read error", zap.String("path", f.config.Path), zap.Error(err))
			return err
		}
		f.offset += int64(n)
		resp.HasMore = f.offset < fileLength

		var lines []string
		if f.consumeBytes(buf[:n], func(line string) {
			if DiscardLineWithZeroBytes && strings.Count(line, "\u0000") >= DiscardZeroBytesThreshold {
				resp.HasBroken = true
			} else {
				lines = append(lines, line)
			}
		}) {
			resp.HasBroken = true
		}
		resp.Lines = lines
	}

	resp.Range = fmt.Sprintf("%d:%d:%d", f.inode, beginOffset, f.offset)
	resp.Bytes = f.offset - beginOffset
	resp.Count = len(resp.Lines)

	if resp.HasMore {
		return nil
	}

	if f.fileChanged {
		f.closeFile()
		resp.HasMore = true
		f.ensureOpened(false)
	}

	return nil
}

func (f *fileSubLogStream) ensureOpened(seekEnd bool) error {
	if f.file != nil {
		return nil
	}

	file, err := os.OpenFile(f.config.Path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	filestat, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	off := int64(0)
	if seekEnd && filestat.Size() > 0 {
		if off, err = file.Seek(-1, io.SeekEnd); err != nil {
			logger.Errorz("[logstream] seek error", zap.Error(err))
			file.Close()
			return err
		}
	}

	f.file = file
	f.filestat = filestat
	f.inode = utils.GetInode(filestat)
	f.offset = off
	f.ignoreFirstLine = off > 0
	return nil
}

func (f *fileSubLogStream) closeFile() {
	if f.file != nil {
		f.file.Close()
	}
	f.file = nil
	f.filestat = nil
	f.offset = 0
	f.fileChanged = false
	f.lineBuffer.Clear()
	f.ignoreFirstLine = false
}

func (f *fileSubLogStream) consumeBytes(b []byte, fun func(string)) bool {
	f.lineBuffer.Add(b)

	for {
		lineBytes, ok := f.lineBuffer.Next()
		if !ok {
			if f.lineBuffer.IsBroken() {
				return true
			}
			return false
		}
		if f.ignoreFirstLine {
			f.ignoreFirstLine = false
			continue
		}
		fun(string(lineBytes))
	}
}
