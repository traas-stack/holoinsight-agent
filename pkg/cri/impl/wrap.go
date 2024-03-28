/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package impl

import (
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"os"
	"time"
)

var (
	timeout = "180"
)

func init() {
	s := os.Getenv("CRI_EXEC_TIMEOUT")
	if x := cast.ToInt(s); x > 0 {
		timeout = s
	}
}

// wrapTimeout wraps cmd with timeout -s KILL <seconds> to prevent the process from hanging and not exiting for any reason.
func wrapTimeout(c *cri.Container, cmd []string) []string {
	// Note:
	// Different busybox versions have different timeout command formats
	// In alpined based container, timeout will generate zombie processes
	// timeout -s KILL <seconds> cmd...
	// return append([]string{"timeout", "-s", "SIGKILL", timeout}, cmd...)
	if c.Pid1CanRecycleZombieProcesses {
		return append([]string{core.BusyboxPath, "timeout", "-s", "SIGKILL", timeout}, cmd...)
	}
	return cmd
}

// wrapEnv wraps envs with _FROM=holoinsight-agent. This env is used to mark the source of the call.
func wrapEnv(envs []string) []string {
	return append(envs, "_FROM=holoinsight-agent", "_TS="+cast.ToString(time.Now().UnixMilli()))
}
