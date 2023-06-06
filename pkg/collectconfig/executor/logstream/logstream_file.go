/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/utils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	fileEndModeNone fileEndMode = iota
	fileEndModeChanged
	fileEndModeDeleted
	fileEndModePermission
	fileEndModeError
)

var (
	DefaultLogInputConfig = LogInputConfig{
		MaxLineSize:    1024 * 1024,
		MaxIOReadBytes: 4 * 1024 * 1024,
	}
	TruncatedErr = errors.New("truncated")
)

type (
	fileEndMode    uint8
	LogInputConfig struct {
		Path           string
		MaxLineSize    int
		MaxIOReadBytes int64
	}
	// FileLogStream 将一个路径(不可变)映射为一个日志流
	FileLogStream struct {
		config    LogInputConfig
		mutex     sync.Mutex
		listeners []Listener
		fileLogStreamState
	}
	fileLogStreamState struct {
		file            *os.File
		offset          int64
		cache           sync.Map
		cursor          int64
		lineBuffer      *utils.LineBuffer
		ignoreFirstLine bool
		fileEndMode     fileEndMode
		fileEndSize     int64
		continued       bool
		//是否是第一次pull
		firstPull bool
		// only *inx
		inode               uint64
		pendingBytes        int64
		pendingReads        int32
		matchesSuccessCount int
	}
	fileLogStreamStateObj struct {
		File            string
		Offset          int64
		Cache           map[int64]*ReadResponse
		Cursor          int64
		LineBuffer      *utils.LineBufferStateObj
		IgnoreFirstLine bool
		FileEndMode     fileEndMode
		FileEndSize     int64
		Continued       bool
		FirstPull       bool
		Inode           uint64
		PendingBytes    int64
		PendingReads    int32
	}
)

func (f *FileLogStream) fileId() string {
	return fmt.Sprintf("ino:%d", f.inode)
}

func (f *FileLogStream) LoadReadState(s *LoadReadState) error {
	if s.Cursor > f.cursor {
		return fmt.Errorf("invalid cursor now=[%d] requested=[%d]", f.cursor, s.Cursor)
	}
	if s.Cursor != f.cursor {
		cached, ok := f.cache.Load(s.Cursor)
		if !ok {
			return fmt.Errorf("no cache for cursor now=[%d] requested=[%d]", f.cursor, s.Cursor)
		}
		resp := cached.(*ReadResponse)
		if resp.FileId != s.FileId && resp.BeginOffset != s.Offset {
			return fmt.Errorf("invalid cursor cache, cached=[%s/%s] requested=[%s/%d]", resp.FileId, resp.BeginOffset, s.FileId, s.Offset)
		}
	} else {
		if !(f.inode == 0 && s.FileId == "") && f.fileId() != s.FileId {
			return fmt.Errorf("inode mismatch now=[%s] requested=[%s]", f.fileId(), s.FileId)
		}
		if s.Offset > f.offset {
			return errors.New("invalid offset")
		}
	}

	f.cache.Range(func(key, value any) bool {
		cursor := key.(int64)
		if cursor >= s.Cursor {
			value.(*ReadResponse).remainCount++
		}
		return true
	})

	f.matchesSuccessCount++
	return nil
}

func NewFileLogStream(config LogInputConfig) *FileLogStream {
	if config.MaxIOReadBytes < DefaultLogInputConfig.MaxIOReadBytes {
		config.MaxIOReadBytes = DefaultLogInputConfig.MaxIOReadBytes
	}
	if config.MaxLineSize < DefaultLogInputConfig.MaxLineSize {
		config.MaxLineSize = DefaultLogInputConfig.MaxLineSize
	}
	return &FileLogStream{
		config: config,
		fileLogStreamState: fileLogStreamState{
			lineBuffer: utils.NewLineBuffer(config.MaxLineSize),
		},
	}
}

func (f *FileLogStream) Start() {
}

func (f *FileLogStream) Stop() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.closeFile()
}

func init() {
	gob.Register(&fileLogStreamStateObj{})
}

func (f *FileLogStream) LoadState(state *fileLogStreamStateObj) error {
	if err := f.ensureOpened(true); err != nil {
		return err
	}

	if f.inode != state.Inode {
		return fmt.Errorf("file changed oldInode=[%d] newInode=[%d]", state.Inode, f.inode)
	}

	if _, err := f.file.Seek(state.Offset, io.SeekStart); err != nil {
		f.closeFile()
		return err
	}

	f.offset = state.Offset
	f.cursor = state.Cursor
	f.lineBuffer.LoadState(state.LineBuffer)
	f.ignoreFirstLine = state.IgnoreFirstLine
	f.fileEndMode = state.FileEndMode
	f.fileEndSize = state.FileEndSize
	f.continued = state.Continued
	f.firstPull = state.Continued
	f.inode = state.Inode
	f.pendingBytes = state.PendingBytes
	f.pendingReads = state.PendingReads

	for key, response := range state.Cache {
		f.cache.Store(key, response)
	}

	return nil
}

func (f *FileLogStream) SaveState() (*fileLogStreamStateObj, error) {
	if f.file == nil {
		return nil, nil
	}

	if !f.lineBuffer.Empty() {
		return nil, errors.New("line buffer is not empty")
	}

	cache := make(map[int64]*ReadResponse)
	f.cache.Range(func(key, value any) bool {
		cache[key.(int64)] = value.(*ReadResponse)
		return true
	})

	state := &fileLogStreamStateObj{
		File:            f.config.Path,
		Offset:          f.offset,
		Cache:           cache,
		Cursor:          f.cursor,
		LineBuffer:      f.lineBuffer.SaveState(),
		IgnoreFirstLine: f.ignoreFirstLine,
		FileEndMode:     f.fileEndMode,
		FileEndSize:     f.fileEndSize,
		Continued:       f.continued,
		FirstPull:       f.firstPull,
		Inode:           f.inode,
		PendingBytes:    f.pendingBytes,
		PendingReads:    f.pendingReads,
	}

	return state, nil
}

func (f *FileLogStream) getCache(cursor int64) *ReadResponse {
	if c, ok := f.cache.Load(cursor); ok {
		return c.(*ReadResponse)
	} else {
		return nil
	}
}

// 读一次日志, 读的结果会关联到一个cursor上;
// 如果当前请求里的cursor落后了, 那么意味着它将要读取的cursor之前已经被读过了, 此时放在缓存里, 取缓存即可.
// 如果当前请求里的cursor==下一个期望读的cursor, 那么执行一次真正的IO读, 并将结果关联到该cursor上, 增加下一个期望读的cursor
// 返回 @resp 对应的一次结果
// 返回 @cursor 对应的一次结果
// 返回 @err 遇到的错误, 一旦遇到错误, 这个流就应该被弃用了, 严重错误
// 调用方检查返回结果, err!=nil 时候代表发生严重错误(一般是没有遵守约定), 此时resp一定是nil,调用方必须调整cursor=最新cursor(第二个返回值), 然后重新发起调用
func (f *FileLogStream) Read(request *ReadRequest) (*ReadResponse, int64, error) {
	reqCursor := request.Cursor

	// check cache
	if c := f.getCache(reqCursor); c != nil {
		if atomic.AddInt32(&c.remainCount, -1) == 0 {
			f.cache.Delete(reqCursor)
			atomic.AddInt64(&f.pendingBytes, -c.Bytes())
			atomic.AddInt32(&f.pendingReads, -1)
		}

		fc := atomic.LoadInt64(&f.cursor)
		if c.NextCursor < fc {
			c.HasMore = true
		}
		return c, fc, nil
	}

	// 加锁保证只有一个goroutine能执行到这里

	f.mutex.Lock()
	defer f.mutex.Unlock()

	// double check cache
	if c := f.getCache(reqCursor); c != nil {
		if atomic.AddInt32(&c.remainCount, -1) == 0 {
			f.cache.Delete(reqCursor)
			atomic.AddInt64(&f.pendingBytes, -c.Bytes())
			atomic.AddInt32(&f.pendingReads, -1)
		}
		return c, f.cursor, nil
	}

	if reqCursor != f.cursor {
		// 这说明 用户传入了一个 stale cursor
		// 此时只能让用户跳转到最新的cursor上
		return nil, f.cursor, fmt.Errorf("stale cursor request=[%d] current=[%d]", reqCursor, f.cursor)
	}

	resp := f.read()
	// 大于0才有存下的意义
	if resp.remainCount > 0 {
		atomic.AddInt64(&f.pendingBytes, resp.Bytes())
		atomic.AddInt32(&f.pendingReads, 1)
		f.cache.Store(resp.Cursor, resp)
	}
	// 这个地方 volatile write 是为了让上面能在锁外read到
	atomic.StoreInt64(&f.cursor, f.cursor+1)

	// 通知变化
	f.fire(f.cursor - 1)

	return resp, f.cursor, nil
}

func (f *FileLogStream) read() *ReadResponse {
	resp := &ReadResponse{
		Cursor:      f.cursor,
		NextCursor:  f.cursor + 1,
		IOStartTime: time.Now(),
		// 此时我们在'写锁'内, 修改监听器需要修改'写锁', 该cursor可以被消费的次数等于监听器数
		// 考虑到当前goroutine一定会消费该数据, 于是这里有一个 '-1'
		remainCount: int32(len(f.listeners) - 1),
	}

	err := f.ensureOpened(true)
	if err != nil {
		resp.Error = err
		return resp
	}

	fileStat, err := f.file.Stat()
	if err != nil {
		resp.Error = err
		f.closeFile()
		return resp
	}

	if f.fileEndMode == fileEndModeNone {
		newStat, err := os.Stat(f.config.Path)
		if err != nil {
			if os.IsNotExist(err) {
				// 文件不存在了
				f.fileEndMode = fileEndModeDeleted
				logger.Infoz("file deleted", //
					zap.String("path", f.config.Path),
					zap.Uint64("inode", f.inode)) //

			} else if os.IsPermission(err) {
				// 没权限读
				f.fileEndMode = fileEndModePermission
				logger.Infoz("file unreadable", //
					zap.String("path", f.config.Path), //
					zap.Uint64("inode", f.inode),      //
					zap.Error(err))                    //

			} else {
				// other exception
				f.fileEndMode = fileEndModeError
				logger.Infoz("file error", //
					zap.String("path", f.config.Path), //
					zap.Uint64("inode", f.inode),      //
					zap.Error(err))                    //
			}
		} else if !os.SameFile(newStat, fileStat) {
			f.fileEndMode = fileEndModeChanged
			// 通过这个技巧可以获取移动后的文件名 不知道底层是怎么实现的
			movedTo, _ := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", f.file.Fd()))

			logger.Infoz("file changed",
				zap.String("path", f.config.Path),      //文件路径
				zap.String("to", movedTo),              // 滚动后的文件
				zap.Int64("fileSize", fileStat.Size()), //
				zap.Uint64("inode", f.inode))           // 此时的文件大小
		}

		if f.fileEndMode != fileEndModeNone {
			// 当前文件消费到完
			f.fileEndSize = fileStat.Size()
			logger.Infoz("enter file end mode",
				zap.String("path", f.config.Path),       //文件路径
				zap.Uint8("mode", uint8(f.fileEndMode)), //
				zap.Int64("fileSize", f.fileEndSize),
				zap.Uint64("inode", f.inode)) //
		}

	}

	var fileLength int64
	if f.fileEndMode != fileEndModeNone {
		fileLength = f.fileEndSize
	} else {
		fileLength = fileStat.Size()
	}

	if fileLength < f.offset {
		// truncated
		resp.Error = TruncatedErr
		f.closeFile()
		return resp
	}

	resp.BeginOffset = f.offset
	if f.offset < fileLength {
		readBytes := fileLength - f.offset
		if readBytes > f.config.MaxIOReadBytes {
			readBytes = f.config.MaxIOReadBytes
		}
		buf := make([]byte, readBytes)
		n, err := f.file.Read(buf)
		if err != nil {
			if err != io.EOF {
				resp.Error = err
				f.closeFile()
				return resp
			}
		}
		buf = buf[:n]
		f.offset += int64(n)
		resp.EndOffset = f.offset
		resp.HasMore = f.offset < fileLength

		var lines []string
		//  lines := make([]string, 0, i.lineBuffer.AvailableLines())
		if f.consumeBytes(buf[:n], func(line string) {
			lines = append(lines, line)
		}) {
			resp.HasBroken = true
		}
		resp.Lines = lines
	} else {
		resp.EndOffset = f.offset
	}

	resp.Continued = f.continued
	f.continued = true
	if f.firstPull {
		resp.Continued = true
		f.firstPull = false
	}

	resp.FileId = fmt.Sprintf("ino:%d", f.inode)
	resp.Path = f.config.Path
	resp.HasBuffer = !f.lineBuffer.Empty()

	if resp.HasMore {
		return resp
	}

	// 读完内容了
	if f.fileEndMode != fileEndModeNone {
		mode := f.fileEndMode
		// closeFile会清掉mode, 所以要先记录下来
		f.closeFile()

		switch mode {
		// 文件轮转case
		case fileEndModeChanged:
			// 此处不关心结果
			if err := f.ensureOpened(false); err != nil {
				f.continued = false
			} else {
				// 立即读一次文件
				f.continued = true
				resp.HasMore = true
			}
		case fileEndModeDeleted:
			// 文件被删除, 没必要重新open, 等下次就行
			f.continued = false
		case fileEndModePermission:
			// 文件没权限, 没必要重新open, 等下次就行
			f.continued = false
		case fileEndModeError:
			// 文件其他错误, 没必要重新open, 等下次就行
			f.continued = false
		}
	}

	return resp
}

func (f *FileLogStream) ensureOpened(seekEnd bool) error {
	if f.file != nil {
		return nil
	}

	file, err := os.Open(f.config.Path)
	if err != nil {
		return err
	}
	f.ignoreFirstLine = false
	f.offset = 0
	fileStat, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	f.inode = utils.GetInode(fileStat)

	var off int64
	if seekEnd {
		if fileStat.Size() == 0 {
			off, err = file.Seek(0, io.SeekEnd)
		} else {
			off, err = file.Seek(-1, io.SeekEnd)
		}
	} else {
		off, err = file.Seek(0, io.SeekStart)
	}
	if err != nil {
		file.Close()
		return err
	}
	f.file = file
	f.offset = off
	f.ignoreFirstLine = off > 0
	f.lineBuffer.Clear()
	return nil
}

func (f *FileLogStream) closeFile() {
	if f.file != nil {
		f.file.Close()
	}
	f.file = nil
	f.offset = 0
	f.fileEndMode = fileEndModeNone
	f.fileEndSize = 0
	f.lineBuffer.Clear()
	f.ignoreFirstLine = false
	f.continued = false
}

func (f *FileLogStream) consumeBytes(b []byte, fun func(string)) bool {
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
		// TODO deep copy
		line := string(lineBytes)
		fun(line)
	}
}

// AddListener 添加监听器, 添加了监听器之后才能够Read
// 添加监听器 @l, 返回该监听器下一个可用的cursor
func (f *FileLogStream) AddListener(l Listener) int64 {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.listeners = append(f.listeners, l)

	return f.cursor
}

// RemoveListener 移除监听器, 用户必须保证调用该方法之后不会再调用read, 该方法和read也不应该被并发调用
// @l: 监听器
// @cursor: 监听器当前的cursor
func (f *FileLogStream) RemoveListener(l Listener, cursor int64) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// 当移除一个listener时, 需要适当地释放它对未读的cursor的引用, 这样可以避免

	for i, v := range f.listeners {
		if v == l {
			temp := make([]Listener, 0, len(f.listeners)-1)
			temp = append(temp, f.listeners[:i]...)
			temp = append(temp, f.listeners[i+1:]...)

			// 下面这行的本意是尽快释放指针, 但会导致 fire 里出现NPE, 反正我们已经不会再持有 f.listeners 的引用了, 因此让它自然而然 GC 即可
			// f.listeners[len(f.listeners)-1] = nil

			// 我们不会对listeners原地修改, 都是copy-on-write
			// 原因见fire方法
			f.listeners = temp

			// 我们现在在锁内 这个区间 [cursor, f.cursor) 本来是该本我们读取的
			// 但现在我们提前释放了 listener 我们不会再去读取该区间了 因此需要提前释放引用
			for ; cursor < f.cursor; cursor++ {
				if c := f.getCache(cursor); c != nil {
					if atomic.AddInt32(&c.remainCount, -1) == 0 {
						atomic.AddInt64(&f.pendingBytes, -c.Bytes())
						atomic.AddInt32(&f.pendingReads, -1)
						f.cache.Delete(cursor)
					}
				}
			}
			break
		}
	}

}

func (f *FileLogStream) fire(cursor int64) {
	// TODO 解释这里为什么go:
	// 此时我们肯定处于lock中 通知listener change之后, 有可能listener自己发起调用, 可能导致死锁 于是这里go一下

	// TODO 20220727: 我想了一下没必要这么复杂, 现在这个 Changed 方法都还没被用上呢... 直接在当前goroutine里执行即可

	listeners := f.listeners
	// go func() {
	for _, l := range listeners {
		l.Changed(f.config.Path, f, cursor)
	}
	// }()
}

func (f *FileLogStream) Stat() Stat {
	return Stat{
		LatestCursor: f.cursor,
		PendingBytes: atomic.LoadInt64(&f.pendingBytes),
		PendingReads: atomic.LoadInt32(&f.pendingReads),
	}
}

func (f *FileLogStream) Clean() {
	f.mutex.Lock()
	expireTime := time.Now().Add(-3 * time.Minute)
	f.cache.Range(func(key, value interface{}) bool {
		resp := value.(*ReadResponse)
		if resp.IOStartTime.Before(expireTime) {
			// 非常严重的错误
			f.cache.Delete(key)
			atomic.AddInt64(&f.pendingBytes, -resp.Bytes())
			logger.Errorz("clean cursor cache", //
				zap.Int32("remainCount", resp.remainCount), //
				zap.Time("ioStartTime", resp.IOStartTime),  //
				zap.String("path", f.config.Path))          //
		}
		return true
	})
	f.mutex.Unlock()
}
