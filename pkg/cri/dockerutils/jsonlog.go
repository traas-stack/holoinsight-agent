/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package dockerutils

import (
	"encoding/json"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/utils"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

type (
	// docker json log model: https://docs.docker.com/config/containers/logging/json-file/
	DockerLog struct {
		Stream string
		Log    string
		Time   string
	}
)

const (
	// A virtual path that represents the standard output of the container
	DockerJsonLogFile = "/virtual/stdout/json.log"
)

func DecodeJsonLog(line string) (*DockerLog, error) {
	dl := DockerLog{}
	if err := json.Unmarshal(util.ZeroCopyStringToBytes(line), &dl); err != nil {
		return nil, err
	}
	dl.Log = utils.TrimCRLFString(dl.Log)
	return &dl, nil
}
