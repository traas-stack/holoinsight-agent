/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package core

import (
	"os"
	"path/filepath"
)

var HelperToolLocalPath = "/usr/local/holoinsight/agent/bin/helper"
var BusyboxLocalPath = "/usr/local/holoinsight/agent/bin/busybox"
var HelperToolPath = "/tmp/holoinsight/helper"
var BusyboxPath = "/tmp/holoinsight/busybox"

type (
	HelperBaseResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

func init() {
	wd, _ := os.Getwd()
	HelperToolLocalPath = filepath.Join(wd, "/bin/helper")
}
