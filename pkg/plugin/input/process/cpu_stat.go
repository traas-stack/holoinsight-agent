/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package process

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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
