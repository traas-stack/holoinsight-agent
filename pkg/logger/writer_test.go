package logger

import (
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestFilepathExt(t *testing.T) {
	fmt.Println(filepath.Ext("a.log"))
}

func TestLogWriter(t *testing.T) {
	w, err := NewRotateWriter(LogConfig{
		Filename:           "/Users/xzchaoo/logs/a.log",
		MaxSize:            5 * 1024,
		MaxBackupCount:     5,
		TimeLayout:         "2006-01-02_15-04",
		MaxBackupsSize:     200 * 1024,
		DeleteScanPatterns: []string{"/Users/xzchaoo/logs/a.log.*", "/Users/xzchaoo/logs/a-*log"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	for {
		str := fmt.Sprintf("%s INFO sjdlkfjsdklf sdjfds fjasklf sklfjlask fjdlskf\n", time.Now().Format(time.RFC3339))
		w.Write([]byte(str))
		time.Sleep(100 * time.Millisecond)
	}
}

func BenchmarkStruct1(b *testing.B) {
	// https://stackoverflow.com/questions/27775376/value-receiver-vs-pointer-receiver

	var logFiles []*logFile
	today := util.TimeTruncateToDay(time.Now())
	for i := 0; i < 10000; i++ {
		logFiles = append(logFiles, &logFile{
			path:  "a.log",
			time:  today.Add(time.Duration(rand.Intn(86400)) * time.Second),
			index: rand.Int(),
		})
	}

	for i := 0; i < b.N; i++ {
		bak := make([]*logFile, len(logFiles))
		copy(bak, logFiles)
		sort.Sort(&logFileSliceSort{s: bak})
	}
}
