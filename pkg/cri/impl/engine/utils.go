/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package engine

import (
	"github.com/spf13/cast"
	"os"
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
func wrapTimeout(cmd []string) []string {
	// timeout -s KILL <seconds> cmd...
	return append([]string{"timeout", "-s", "KILL", timeout}, cmd...)
}

// wrapEnv wraps envs with _FROM=holoinsight-agent. This env is used to mark the source of the call.
func wrapEnv(envs []string) []string {
	return append(envs, "_FROM=holoinsight-agent")
}
