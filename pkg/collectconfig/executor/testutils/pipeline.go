/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package testutils

import (
	"encoding/json"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	"os"
)

func LoadLogPipeline(confPath string) (*executor.LogPipeline, error) {

	f, err := os.Open(confPath)
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
	pipeline, err := executor.NewPipeline(&api.SubTask{
		ct, sqlTask,
	}, sotrage2, lsm)

	if err != nil {
		return nil, err
	}

	return pipeline, nil
}
