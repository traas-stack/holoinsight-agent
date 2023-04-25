/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package processperf

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type (
	linuxIoStat struct {
		rchar                 int64
		wchar                 int64
		syscr                 int64
		syscw                 int64
		read_bytes            int64
		write_bytes           int64
		cancelled_write_bytes int64
	}
)

func readIoStatus(pid int32) (*linuxIoStat, error) {
	path := fmt.Sprintf("/proc/%d/io", pid)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	scanner.Split(bufio.ScanLines)
	stat := &linuxIoStat{}
	for scanner.Scan() {
		line := scanner.Text()
		index := strings.IndexByte(line, ':')
		if index < 0 {
			continue
		}
		switch line[:index] {
		case "rchar":
			stat.rchar, _ = strconv.ParseInt(line[index+2:], 10, 64)
		case "wchar":
			stat.wchar, _ = strconv.ParseInt(line[index+2:], 10, 64)
		case "syscr":
			stat.syscr, _ = strconv.ParseInt(line[index+2:], 10, 64)
		case "syscw":
			stat.syscw, _ = strconv.ParseInt(line[index+2:], 10, 64)
		case "read_bytes":
			stat.read_bytes, _ = strconv.ParseInt(line[index+2:], 10, 64)
		case "write_bytes":
			stat.write_bytes, _ = strconv.ParseInt(line[index+2:], 10, 64)
		case "cancelled_write_bytes":
			stat.cancelled_write_bytes, _ = strconv.ParseInt(line[index+2:], 10, 64)
		}
	}
	return stat, nil
}
