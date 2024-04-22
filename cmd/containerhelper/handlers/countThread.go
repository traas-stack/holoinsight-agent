/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"bufio"
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"os"
	"strconv"
	"strings"
)

// Count the number of thread
func countThreadHandler(_ string, resp *model.Resp) error {
	pids, err := process.Pids()
	if err != nil {
		return err
	}

	values := make(map[string]interface{})
	totalThreads := readTotalThreads(pids)
	values["process_pids"] = len(pids)
	values["process_threads"] = totalThreads
	resp.Data = values
	return nil
}

// Only runs on Linux
// Get total threads count of OS
func readTotalThreads(pids []int32) int {
	totalThreads := int64(0)
	for _, pid := range pids {
		path := fmt.Sprintf("/proc/%d/status", pid)
		file, err := os.Open(path)
		if err != nil {
			continue
		} else {
			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "Threads:\t") {
					x, err := strconv.ParseInt(line[len("Threads:\t"):], 10, 64)
					if err != nil {
						continue
					} else {
						totalThreads += x
					}
				}
			}
			file.Close()
		}
	}
	return int(totalThreads)
}
