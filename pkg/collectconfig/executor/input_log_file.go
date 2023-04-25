/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/utils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
	"io"
	"os"
)

const (
	fileEndModeNone fileEndMode = iota
	fileEndModeChanged
	fileEndModeDeleted
	fileEndModePermission
	fileEndModeError
)

var (
	DefaultLogInputConfig LogInputConfig
	TruncatedErr          = errors.New("truncated")
)

type (
	fileEndMode    uint8
	LogInputConfig struct {
		Path        string
		MaxLineSize int
		IOReadBytes int64
	}
	// TODO 支持二分搜索
	// FileLogInput 表示一个日志输入来源, 只能被串行化使用
	FileLogInput struct {
		// 一个唯一的key, 可用于日志追踪
		key    string
		config LogInputConfig
		// 消费的offset
		offset          int64
		file            *os.File
		lineBuffer      *utils.LineBuffer
		ignoreFirstLine bool
		fileEndMode     fileEndMode
		fileEndSize     int64
		continued       bool
		//是否是第一次pull
		firstPull bool
		// only *inx
		inode uint64
	}
	PullRequest  struct{}
	PullResponse struct {
		// 这里不会合并多行, 也不会解析时间戳, 需要上层自己去处理
		Lines []string
		// 数据是否是连续的
		// 我们规定第一个PullResult总是连续的
		Continued bool
		// 是否还有更多数据
		HasMore bool

		// 一些内部信息, 可以依赖他们去调试
		HasBroken   bool
		HasBuffer   bool
		BeginOffset int64
		EndOffset   int64
		FileLength  int64
		FileId      string
	}
	// 一个日志组, 单行情况下一行日志为一个组
	LogGroup struct {
		// 第一行
		Line string
		// 多行case, 99%的case都只有一行
		Lines []string
	}
)

func init() {
	DefaultLogInputConfig.IOReadBytes = 4 * 1024 * 1024
	DefaultLogInputConfig.MaxLineSize = 64 * 1024
}

func (l *LogGroup) FirstLine() string {
	return l.Line
}

func (l *LogGroup) Add(line string) {
	if len(l.Lines) == 0 {
		l.Line = line
	}
	l.Lines = append(l.Lines, line)
}

func (l *LogGroup) SetOneLine(line string) {
	l.Line = line
	l.Lines[0] = line
}

func (l *LogGroup) reset() {
	l.Line = ""
	l.Lines = nil
}

func NewLogInput(key string, config LogInputConfig) *FileLogInput {
	if config.MaxLineSize < DefaultLogInputConfig.MaxLineSize {
		config.MaxLineSize = DefaultLogInputConfig.MaxLineSize
	}
	if config.IOReadBytes < DefaultLogInputConfig.IOReadBytes {
		config.IOReadBytes = DefaultLogInputConfig.IOReadBytes
	}
	return &FileLogInput{
		key:        key,
		config:     config,
		lineBuffer: utils.NewLineBuffer(config.MaxLineSize),
		firstPull:  true,
	}
}

func (i *FileLogInput) Start() {
	logger.Infoz("input start", zap.String("key", i.key))
}

func (i *FileLogInput) Stop() {
	i.closeFile()
	logger.Infoz("input stop", zap.String("key", i.key))
}

func (i *FileLogInput) ensureOpened(seekEnd bool) error {
	if i.file == nil {
		file, err := os.Open(i.config.Path)
		if err != nil {
			logger.Errorz("fail to openfile", //
				zap.String("path", i.config.Path))
			return err
		}
		i.ignoreFirstLine = false
		i.offset = 0
		fileStat, err := file.Stat()
		if err != nil {
			file.Close()
			logger.Errorz("fail to seek to end", zap.String("path", i.config.Path))
			return err
		}
		i.inode = utils.GetInode(fileStat)

		// seek to end
		if seekEnd {
			// TODO 解释为什么-1 这里要从vessel的代码说起
			offset, err := file.Seek(-1, io.SeekEnd)
			if err != nil {
				file.Close()
				logger.Errorz("fail to seek to end %s", zap.String("path", i.config.Path))
				return err
			}
			logger.Infoz("[input] open and seek to end", //
				zap.String("path", i.config.Path), //
				zap.Int64("offset", offset),       //
				zap.Uint64("inode", i.inode))      //
			i.offset = offset
			i.ignoreFirstLine = offset > 0
		} else {
			logger.Infoz("[input] open and no seek", //
				zap.String("path", i.config.Path), //
				zap.Uint64("inode", i.inode))      //
		}
		i.file = file
		// 重新打开一次文件后 buffer 都要清掉
		i.lineBuffer.Clear()
	}
	return nil
}

func (i *FileLogInput) closeFile() {
	if i.file != nil {
		i.file = nil
		i.file.Close()
	}
	i.fileEndMode = fileEndModeNone
	i.fileEndSize = 0
	i.offset = 0
	i.lineBuffer.Clear()
	i.ignoreFirstLine = false
	i.continued = false
}

func (i *FileLogInput) consumeBytes(b []byte, f func(string)) bool {
	i.lineBuffer.Add(b)

	for {
		lineBytes, ok := i.lineBuffer.Next()
		if !ok {
			if i.lineBuffer.IsBroken() {
				return true
			}
			return false
		}
		if i.ignoreFirstLine {
			i.ignoreFirstLine = false
			continue
		}
		// TODO deep copy
		line := string(lineBytes)
		f(line)
	}
}

func (i *FileLogInput) Reset() {
	// 复位
	i.closeFile()
}

func (i *FileLogInput) Finished() bool {
	return false
}

func (i *FileLogInput) Pull(req *PullRequest) (*PullResponse, error) {
	// 99%的pull涉及3次IO, 看能否节省一下:
	// 1. os.Stat
	// 2. file.Stat
	// 3. file.Read

	// 确保文件已经打开
	// 99% case 这里直接从文件尾巴开始读即可, 比如程序刚启动时

	err := i.ensureOpened(true)
	if err != nil {
		return nil, err
	}

	// 查询一下当前打开的文件的状态
	fileStat, err := i.file.Stat()
	if err != nil {
		// 出问题了 应该是比较严重的, 此时不能保证数据的平滑了
		i.closeFile()
		logger.Errorz("fail to stat file", zap.Error(err))
		return nil, err
	}

	// 检查文件是否发生变化
	// fileEnd 模式意思是对应路径的文件已经发生了 "某种变化", 具体是哪种变化无所谓
	// 但由于我们已经打开了文件句柄, 于是我们是可以继续读文件的
	// 为了应对文件移动后有其他进程继续往文件里写数据, 导致我们退不出
	// 一旦发现文件发生"某种变化", 我们最多将当前文件读到当前文件大小, 就算结束了
	// 1. 对于文件轮转/删除场景来说或我们的行为是正确的, 不会导致文件句柄泄漏
	if i.fileEndMode == fileEndModeNone {
		newStat, err := os.Stat(i.config.Path)
		if err != nil {
			if os.IsNotExist(err) {
				// 文件不存在了
				i.fileEndMode = fileEndModeDeleted
				logger.Infoz("file deleted", //
					zap.String("path", i.config.Path),
					zap.Uint64("inode", i.inode)) //

			} else if os.IsPermission(err) {
				// 没权限读
				i.fileEndMode = fileEndModePermission
				logger.Infoz("file unreadable", //
					zap.String("path", i.config.Path), //
					zap.Uint64("inode", i.inode),      //
					zap.Error(err))                    //

			} else {
				// other exception
				i.fileEndMode = fileEndModeError
				logger.Infoz("file error", //
					zap.String("path", i.config.Path), //
					zap.Uint64("inode", i.inode),      //
					zap.Error(err))                    //
			}
		} else if !os.SameFile(newStat, fileStat) {
			i.fileEndMode = fileEndModeChanged
			// 通过这个技巧可以获取移动后的文件名 不知道底层是怎么实现的
			movedTo, _ := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", i.file.Fd()))

			logger.Infoz("file changed",
				zap.String("path", i.config.Path),      //文件路径
				zap.String("to", movedTo),              // 滚动后的文件
				zap.Int64("fileSize", fileStat.Size()), //
				zap.Uint64("inode", i.inode))           // 此时的文件大小
		}

		if i.fileEndMode != fileEndModeNone {
			// 当前文件消费到完
			i.fileEndSize = fileStat.Size()
			logger.Infoz("enter file end mode",
				zap.String("path", i.config.Path),       //文件路径
				zap.Uint8("mode", uint8(i.fileEndMode)), //
				zap.Int64("fileSize", i.fileEndSize),
				zap.Uint64("inode", i.inode)) //
		}

	}

	logger.Infoz("[input] file stat", //
		zap.String("path", i.config.Path),              //
		zap.Uint64("inode", i.inode),                   //
		zap.Int64("fileSize", fileStat.Size()),         //
		zap.Int64("offset", i.offset),                  //
		zap.Uint8("fileEndMode", uint8(i.fileEndMode)), //
	)

	var fileLength int64
	if i.fileEndMode != fileEndModeNone {
		fileLength = i.fileEndSize
	} else {
		fileLength = fileStat.Size()
	}

	result := &PullResponse{
		HasMore:     false,
		BeginOffset: i.offset,
		EndOffset:   i.offset,
		FileLength:  fileLength,
	}

	if fileLength < i.offset {
		// 发生 truncate 正确性无法得到任何保障, 于是只能重开
		i.closeFile()
		return nil, TruncatedErr
	}

	if i.offset < fileLength {

		// buf大小不超过最大可readBytes
		readBytes := i.config.IOReadBytes
		if readBytes > fileLength-i.offset {
			readBytes = fileLength - i.offset
		}

		// TODO buf 复用
		buf := make([]byte, readBytes)
		n, err := i.file.ReadAt(buf, i.offset)
		if err != nil {
			if err != io.EOF {
				// 其他错误要关闭文件
				i.closeFile()
				return nil, err
			}
			// EOF
			// logger.Infof("read EOF")
			// EOF是不要紧的
		}
		i.offset += int64(n)
		result.EndOffset = i.offset
		result.HasMore = i.offset < fileLength

		var lines []string
		//  lines := make([]string, 0, i.lineBuffer.AvailableLines())
		if i.consumeBytes(buf[:n], func(line string) {
			lines = append(lines, line)
		}) {
			result.HasBroken = true
		}
		result.Lines = lines

	}

	result.Continued = i.continued
	i.continued = true
	if i.firstPull {
		result.Continued = true
		i.firstPull = false
	}

	result.FileId = fmt.Sprintf("ino:%d", i.inode)
	result.HasBuffer = !i.lineBuffer.Empty()

	if result.HasMore {
		return result, nil
	}

	// 读完内容了
	if i.fileEndMode != fileEndModeNone {
		mode := i.fileEndMode
		// closeFile会清掉mode, 所以要先记录下来
		i.closeFile()

		switch mode {
		// 文件轮转case
		case fileEndModeChanged:
			// 此处不关心结果
			if err := i.ensureOpened(false); err != nil {
				i.continued = false
			} else {
				// 立即读一次文件
				i.continued = true
				result.HasMore = true
			}
		case fileEndModeDeleted:
			// 文件被删除, 没必要重新open, 等下次就行
			i.continued = false
		case fileEndModePermission:
			// 文件没权限, 没必要重新open, 等下次就行
			i.continued = false
		case fileEndModeError:
			// 文件其他错误, 没必要重新open, 等下次就行
			i.continued = false
		}
	}

	return result, nil
}
