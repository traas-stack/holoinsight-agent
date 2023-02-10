package executor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/output/console"
	"os"
	"testing"
	"time"
)

func TestExecutor_simple(t *testing.T) {
	basename := "multiline"
	// log pipeline
	appconfig.SetDev(true)
	logger.DebugEnabled = true

	logger.SetupZapLogger()

	go generateLogs(basename)
	f, err := os.Open("examples/" + basename + "/1.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	sqlTask := &collectconfig.SQLTask{}
	err = json.NewDecoder(f).Decode(sqlTask)
	if err != nil {
		panic(err)
	}

	lsm := logstream.NewManager()
	lsm.Start()

	sotrage2 := storage.NewStorage()
	sotrage2.Start()

	ct := &collecttask.CollectTask{
		Key:     "TEST",
		Version: "TEST",
		Config:  &collecttask.CollectConfig{},
		Target:  &collecttask.CollectTarget{},
	}
	pipeline, err := NewPipeline(&api.SubTask{
		ct, sqlTask,
	}, sotrage2, lsm)
	if err != nil {
		panic(err)
	}

	go pipeline.Start()

	// 例子
	time.Sleep(time.Hour)
}

func generateLogs(basename string) {
	f, err := os.Open("examples/" + basename + "/1.log")
	if err != nil {
		f, err = os.Open("examples/" + basename + "/1.demolog")
	}
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	f.Close()

	out1, err := os.OpenFile("/tmp/1.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	//out2, err := os.OpenFile("/Users/xzchaoo/logs/bbb/1.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		panic(err)
	}
	for {
		buf := &bytes.Buffer{}
		for i, line := range lines {
			if i == 0 {
				fmt.Fprintln(buf, time.Now().Format("2006-01-02 15:04:05"), line[len("2006-01-02 15:04:05 "):])
			} else {
				fmt.Fprintln(buf, line)
			}
		}
		fmt.Fprint(out1, buf.String())
		time.Sleep(time.Second)
	}
}
