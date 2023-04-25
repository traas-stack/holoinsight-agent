/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logger

import (
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultTimeLayout    = "2006-01-02"
	defaultMaxSize       = 1024 * 1024 * 1024
	defaultMaxBackups    = 5
	defaultMaxBackupSize = defaultMaxBackups * defaultMaxSize
)

type (
	// RotateWriter is an io.Writer with ability to rotate files by timer and size like LOG4J in Java.
	// Some codes are copied from 'gopkg.in/natefinch/lumberjack.v2 v2.0.0'.
	// The entire rotation process is not atomic. If a crash happens when rotating, it may leave an intermediate state.
	RotateWriter struct {
		cfg LogConfig
		// size is the size of current writing log file.
		size int64
		// file is the current writing log file. It may be nil if no write.
		file *os.File
		// mu is Mutex used to protected internal state.
		mu sync.Mutex

		fileOpenTime    time.Time
		forceRotateTime time.Time
		stop            chan struct{}
		timeHelper      timeHelper
	}
	LogConfig struct {
		// Filename is the file to write logs to.  Backup log files will be retained
		// in the same directory.  It uses format '${filename}.${date}.${index}'
		Filename string
		// MaxSize is the max size of a log file before being rotated.
		MaxSize int64
		// MaxBackupCount is the max count of backup files.
		// Backup files are files with name format '${filename}.${date}.${index}'.
		// Only recent MaxBackupCount log files will be retained.
		MaxBackupCount int
		// MaxBackupsSize is the max accumulated size of backup files.
		// Only recent N log files whose accumulated size <= MaxBackupsSize will be retained.
		MaxBackupsSize int64
		// TimeLayout is the layout used when make a backup log files.
		TimeLayout string
		// Remove unknown files that match '${filename}.*' but '${date}' part is not matched to timeLayout
		// For example, 'a.log.2022-01-01_17.1' is not matched to '2006-01-02' pattern. This happens after changing to the timeLayout.
		RemoveUnknownFiles bool
		// DeleteScanPatterns are patterns to scan when search and delete backups.
		DeleteScanPatterns []string
		// FileIndex is used to control index of backup files.
		// When set to 'min' , a.log.${time}.${index} will be rename to a.log.${time}.${index+1}, and a.log will be rename to a.log.${time}.1
		// When set to 'min' , a.log will be rename to a.log.${time}.${maxIndex+1}
		// see https://logging.apache.org/log4j/2.x/manual/appenders.html
		// Defaults to 'min'
		FileIndex FileIndex
	}
	// timeHelper is used to calc align time.
	timeHelper time.Duration
	// logFile encapsulates a log file. For example '/path/to/a.log.${time}.${index}'.
	logFile struct {
		// path is the first part in log file basename
		path string
		// time is the second part in log file basename
		time time.Time
		// index is the third part in log file basename
		index int
	}
	logFileSliceSort struct {
		s        []*logFile
		dateAsc  bool
		indexAsc bool
	}
	fileInfo struct {
		path string
		stat fs.FileInfo
	}
	fileInfoSortedByModDesc []*fileInfo
	FileIndex               uint8
)

const (
	FileIndexMin FileIndex = iota
	FileIndexMax
)

func NewRotateWriter(cfg LogConfig) (*RotateWriter, error) {
	// fix defaults
	if cfg.TimeLayout == "" {
		cfg.TimeLayout = defaultTimeLayout
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = defaultMaxSize
	}
	if cfg.MaxBackupCount <= 0 {
		cfg.MaxBackupCount = defaultMaxBackups
	}
	if cfg.MaxBackupsSize <= 0 {
		cfg.MaxBackupsSize = defaultMaxBackupSize
	}
	{
		expected := cfg.Filename + ".*"
		if !util.StringSliceContains(cfg.DeleteScanPatterns, expected) {
			cfg.DeleteScanPatterns = append(cfg.DeleteScanPatterns, expected)
		}
	}

	w := &RotateWriter{
		cfg: cfg,
		// rotate may occur when 'w.openExistingOrNew', so here initialize fileOpenTime to time.Now()
		fileOpenTime: time.Now(),
		stop:         make(chan struct{}),
		timeHelper:   parseTimeHelper(cfg.TimeLayout),
	}

	// open right now
	err := w.openExistingOrNew(0)
	if err != nil {
		w.Close()
		return nil, err
	}

	go w.rotateLoop()

	return w, nil
}

// rotateLoop use a timer to drive rotation by time. This ensures that file scrolling occurs even if no log is printed.
func (w *RotateWriter) rotateLoop() {
	timer := time.NewTimer(0)
	select {
	case <-timer.C:
	}
	timer.Stop()

	defer timer.Stop()

	now := time.Now()
	next := w.timeHelper.nextAlign(now)
	timer.Reset(next.Sub(now))
	for {
		select {
		case <-w.stop:
			return
		case <-timer.C:
			w.maybeRotate()

			now := time.Now()
			next := w.timeHelper.nextAlign(now)
			timer.Reset(next.Sub(now))
		}
	}
}

// isStop checks if Close is called
func (w *RotateWriter) isStop() bool {
	select {
	case <-w.stop:
		return true
	default:
		return false
	}
}

// maybeRotate trigger a file-rotation by check time
func (w *RotateWriter) maybeRotate() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isStop() {
		return
	}

	if w.fileOpenTime.Before(w.forceRotateTime) {
		// TODO error handle
		w.rotate(false)
	}
}

func (w *RotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	writeLen := int64(len(p))
	if writeLen > w.cfg.MaxSize {
		return 0, fmt.Errorf("write length %d exceeds maximum file size %d", writeLen, w.cfg.MaxSize)
	}

	if w.file == nil {
		if err = w.openExistingOrNew(len(p)); err != nil {
			return 0, err
		}
	}

	if w.size+writeLen > w.cfg.MaxSize {
		if err := w.rotate(true); err != nil {
			return 0, err
		}
	}

	n, err = w.file.Write(p)
	w.size += int64(n)

	return n, err
}

func (w *RotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	close(w.stop)

	return w.closeFile()
}

// openExistingOrNew opens the logfile if it exists and if the current write
// would not put it over MaxSize.  If there is no such file or the write would
// put it over the MaxSize, a new file is created.
func (w *RotateWriter) openExistingOrNew(writeLen int) error {
	info, err := os.Stat(w.cfg.Filename)

	if os.IsNotExist(err) {
		// file is not exist, open new
		return w.openNew()
	}
	if err != nil {
		return fmt.Errorf("error getting log file info: %s", err)
	}

	// This write will exceed the size limit, trigger a rotation
	if info.Size()+int64(writeLen) >= w.cfg.MaxSize {
		return w.rotate(true)
	}

	// Remaining capacity is sufficient, just open it
	file, err := os.OpenFile(w.cfg.Filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// 1. no write permission to w.cfg.filename
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "no write permission to %s: %+v\n", w.cfg.Filename, err)
			return err
		}

		// 2. maybe file deleted by other process/thread just now, just reopen it
		if os.IsNotExist(err) {
			return w.openNew()
		}

		return err
	}
	w.file = file
	w.size = info.Size()
	now := time.Now()
	w.fileOpenTime = now
	w.forceRotateTime = w.timeHelper.nextAlign(now)
	return nil
}

// openNew opens a new log file for writing, moving any old log file out of the
// way.  This method assumes the file has already been closed.
func (w *RotateWriter) openNew() error {
	err := os.MkdirAll(filepath.Dir(w.cfg.Filename), 0744)
	if err != nil {
		return fmt.Errorf("can't make directories for new logfile: %s", err)
	}

	name := w.cfg.Filename
	_, err = os.Stat(name)
	if err == nil {
		// file exists
		return errors.New("file already exists")
	}

	now := time.Now()
	// we use truncate here because this should only get called when we've moved
	// the file ourselves. if someone else creates the file in the meantime,
	// just wipe out the contents.
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}
	w.fileOpenTime = now
	w.forceRotateTime = w.timeHelper.nextAlign(now)
	w.file = f
	w.size = 0
	return nil
}

func (w *RotateWriter) buildFileName(t time.Time, index int) string {
	return fmt.Sprintf("%s.%s.%d", w.cfg.Filename, t.Format(w.cfg.TimeLayout), index)
}

func (w *RotateWriter) rotate(useNowTime bool) error {
	// close current file
	if err := w.closeFile(); err != nil {
		return err
	}

	if err := w.rotate0(useNowTime); err != nil {
		return err
	}

	if err := w.openNew(); err != nil {
		return err
	}

	return nil
}

func (w *RotateWriter) handleUnknownFile(path string) {
	if w.cfg.RemoveUnknownFiles {
		// fmt.Printf("remove unknown file: %s\n", path)
		os.Remove(path)
	}
}

// scanFiles scans files that match "filename.*"
// result is un-ordered
func (w *RotateWriter) scanFiles(filename string) ([]*logFile, error) {
	paths, err := filepath.Glob(filename + ".*")
	if err != nil {
		return nil, err
	}
	var logFiles []*logFile
	for _, path := range paths {
		// a.log.2022-11-11.1
		base := filepath.Base(path)

		index1 := strings.LastIndexByte(base, '.')
		if index1 < 0 {
			w.handleUnknownFile(path)
			continue
		}

		indexStr := base[index1+1:]

		index, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil || index <= 0 {
			w.handleUnknownFile(path)
			continue
		}

		index0 := strings.LastIndexByte(base[:index1], '.')
		if index0 < 0 {
			w.handleUnknownFile(path)
			continue
		}
		timeStr := base[index0+1 : index1]
		parsedTime, err := time.ParseInLocation(w.cfg.TimeLayout, timeStr, time.Local)
		if err != nil {
			w.handleUnknownFile(path)
			continue
		}
		logFiles = append(logFiles, &logFile{
			path:  path,
			time:  parsedTime,
			index: int(index),
		})
	}
	return logFiles, nil
}

func (w *RotateWriter) backupName(name string) string {
	return name
}

func (w *RotateWriter) closeFile() error {
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.size = 0
	return err
}

func (w *RotateWriter) rotate0(useNowTime bool) error {
	// 1. read all log files (filter out those unknown files)
	logFiles, err := w.scanFiles(w.cfg.Filename)
	if err != nil {
		return err
	}

	// 2. move a.log.n -> a.log.${yyyy-MM-dd}.n+1
	var formatTime time.Time
	if useNowTime {
		formatTime = w.timeHelper.truncate(time.Now())
	} else {
		formatTime = w.timeHelper.truncate(w.fileOpenTime)
	}
	var currentPeriodLogFiles []*logFile
	for _, file := range logFiles {
		if file.time == formatTime {
			currentPeriodLogFiles = append(currentPeriodLogFiles, file)
		}
	}

	if w.cfg.FileIndex == FileIndexMin {
		sort.Sort(&logFileSliceSort{s: currentPeriodLogFiles, dateAsc: true, indexAsc: false})
		for _, file := range currentPeriodLogFiles {
			oldPath := file.path
			newPath := w.buildFileName(file.time, file.index+1)
			err := os.Rename(oldPath, newPath)
			if err != nil {
				continue
			}
			file.path = newPath
			file.index++
		}
		// 3. move a.log -> a.log.${yyyy-MM-dd}.1
		newPath := w.buildFileName(formatTime, 1)
		if err := os.Rename(w.cfg.Filename, newPath); err != nil {
			return err
		}
		// add a.log.${formatTime}.1 to logFiles
		logFiles = append(logFiles, &logFile{path: newPath, time: formatTime, index: 1})
	} else {
		currentPeriodMaxIndex := 0
		for _, file := range currentPeriodLogFiles {
			if currentPeriodMaxIndex < file.index {
				currentPeriodMaxIndex = file.index
			}
		}
		// 3. move a.log -> a.log.${formatTime}.${currentPeriodMaxIndex+1}
		newPath := w.buildFileName(formatTime, currentPeriodMaxIndex+1)
		if err := os.Rename(w.cfg.Filename, newPath); err != nil {
			return err
		}
		// add a.log.${formatTime}.${currentPeriodMaxIndex+1} to logFiles
		logFiles = append(logFiles, &logFile{path: newPath, time: formatTime, index: currentPeriodMaxIndex + 1})
	}

	// 4. scan all-related files and sort by modified time desc
	scanFilesMap := make(map[string]struct{})
	for _, pattern := range w.cfg.DeleteScanPatterns {
		if matches, err := filepath.Glob(pattern); err == nil {
			for _, path := range matches {
				scanFilesMap[path] = struct{}{}
			}
		}
	}
	scanFiles := make([]*fileInfo, 0, len(scanFilesMap))
	for path := range scanFilesMap {
		if stat, err := os.Stat(path); err == nil {
			scanFiles = append(scanFiles, &fileInfo{path: path, stat: stat})
		}
	}
	sort.Sort(fileInfoSortedByModDesc(scanFiles))
	accSize := int64(0)
	lastKeepIndex := -1
	for i, file := range scanFiles {
		if i >= w.cfg.MaxBackupCount {
			break
		}
		accSize += file.stat.Size()
		if accSize > w.cfg.MaxBackupsSize {
			break
		} else {
			lastKeepIndex = i
		}
	}
	// delete files between [lastKeepIndex+1, len(scanFiles))
	for i := lastKeepIndex + 1; i < len(scanFiles); i++ {
		os.Remove(scanFiles[i].path)
	}

	return nil
}

func parseTimeHelper(timeLayout string) timeHelper {
	demoTime := time.Date(2006, 1, 2, 0, 0, 0, 0, time.Local)
	demoTimeStr := demoTime.Format(timeLayout)

	for _, interval := range []time.Duration{time.Minute, time.Hour} {
		nextAlignTime := demoTime.Add(interval)
		if nextAlignTimeStr := nextAlignTime.Format(timeLayout); demoTimeStr != nextAlignTimeStr {
			// check valid layout
			if parsed, err := time.ParseInLocation(timeLayout, nextAlignTimeStr, time.Local); err == nil && parsed == nextAlignTime {
				return timeHelper(interval)
			}
		}
	}

	return timeHelper(0)
}

func (h timeHelper) truncate(t time.Time) time.Time {
	if h == 0 {
		return util.TimeTruncateToDay(t)
	}
	return t.Truncate(time.Duration(h))
}

func (h timeHelper) nextAlign(t time.Time) time.Time {
	if h == 0 {
		return util.TimeTruncateToDay(t).AddDate(0, 0, 1)
	}
	return t.Truncate(time.Duration(h)).Add(time.Duration(h))
}

func (f fileInfoSortedByModDesc) Len() int {
	return len(f)
}

func (f fileInfoSortedByModDesc) Less(i, j int) bool {
	return f[i].stat.ModTime().After(f[j].stat.ModTime())
}

func (f fileInfoSortedByModDesc) Swap(i, j int) {
	t := f[i]
	f[i] = f[j]
	f[j] = t
}
func (s *logFileSliceSort) Len() int {
	return len(s.s)
}

func (s *logFileSliceSort) Less(i, j int) bool {
	x := s.s[i]
	y := s.s[j]
	if x.time.Before(y.time) {
		return s.dateAsc
	}
	if y.time.Before(x.time) {
		return !s.dateAsc
	}
	if s.indexAsc {
		return x.index < y.index
	} else {
		return y.index < x.index
	}
}

func (s *logFileSliceSort) Swap(i, j int) {
	t := s.s[i]
	s.s[i] = s.s[j]
	s.s[j] = t
}
