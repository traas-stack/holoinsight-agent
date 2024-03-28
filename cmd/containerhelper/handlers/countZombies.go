/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	countLimit = 100
)

// Count the number of zombie processes
func countZombiesHandler(_ string, resp *model.Resp) error {
	count := 0
	filepath.Walk("/proc", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if _, err := strconv.ParseInt(info.Name(), 10, 32); err != nil {
			return nil
		}
		content, err := os.ReadFile(filepath.Join(path, "/stat"))
		if err != nil {
			return nil
		}
		if strings.Contains(string(content), ") Z 1 ") {
			count++
		}
		if count > countLimit {
			return filepath.SkipAll
		}
		return filepath.SkipDir
	})
	resp.Data = count
	return nil
}
