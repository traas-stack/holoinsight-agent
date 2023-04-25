/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package json

import (
	"bufio"
	"bytes"
	json2 "encoding/json"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/output/all"
	"os"
	"testing"
	"time"
)

func TestExecutor_simple(t *testing.T) {
	// log pipeline
	appconfig.SetDev(true)
	logger.DebugEnabled = true

	//basename := "json"
	logger.SetupZapLogger()

	f, err := os.Open("1.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	sqlTask := &collectconfig.SQLTask{}
	err = json2.NewDecoder(f).Decode(sqlTask)
	if err != nil {
		panic(err)
	}

	go generateLogs(sqlTask)

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
	pipeline, err := executor.NewPipeline(&api.SubTask{
		CT: ct, SqlTask: sqlTask,
	}, sotrage2, lsm)
	if err != nil {
		panic(err)
	}

	go pipeline.Start()

	time.Sleep(time.Hour)
}

func generateLogs(sqltask *collectconfig.SQLTask) {
	f, err := os.Open("1.log")
	if err != nil {
		f, err = os.Open("1.demolog")
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
	out1, err := os.OpenFile(sqltask.From.Log.Path[0].Pattern, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	for {
		buf := &bytes.Buffer{}
		for _, line := range lines {
			m := make(map[string]interface{})
			json2.Unmarshal([]byte(line), &m)
			m["time"] = time.Now().Format(sqltask.From.Log.Time.Layout)
			bs, _ := json2.Marshal(m)
			fmt.Fprintln(buf, string(bs))
		}
		fmt.Fprint(out1, buf.String())
		time.Sleep(time.Second)
	}
}
