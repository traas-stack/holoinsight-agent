package core

import (
	"os"
	"path/filepath"
)

var HelperToolLocalPath = "/usr/local/holoinsight/agent/bin/helper"
var HelperToolPath = "/usr/local/holoinsight/agent/tools/helper"

type (
	HelperBaseResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

func init() {
	wd, _ := os.Getwd()
	HelperToolLocalPath = filepath.Join(wd, "/bin/helper")
	HelperToolPath = filepath.Join(wd, "/tools/helper")
}
