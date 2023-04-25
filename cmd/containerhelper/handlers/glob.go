/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"os"
	"path/filepath"
)

func globHandler(action string, resp *model.Resp) error {
	pattern := os.Args[2]
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	resp.Data = files
	resp.Success = true
	return nil
}
