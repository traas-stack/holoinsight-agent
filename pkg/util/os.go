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

// CreateDirIfNotExists creates dir if it does not exist
func CreateDirIfNotExists(dir string, perm os.FileMode) error {
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, perm)
		}
		return err
	}
	if stat.IsDir() {
		return nil
	}
	return os.ErrExist
}
