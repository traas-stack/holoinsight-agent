/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"os"
	"runtime"
)

var isLinux = runtime.GOOS == "linux"

func IsLinux() bool {
	return isLinux
}

func GetEnvOrDefault(name, defaultValue string) string {
	s := os.Getenv(name)
	if s == "" {
		s = defaultValue
	}
	return s
}
