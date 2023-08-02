//go:build linux

/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"golang.org/x/sys/unix"
	"os"
)

func OpenFileReadonly(file string) (*os.File, error) {
	return os.OpenFile(file, os.O_RDONLY|unix.O_NOATIME, 0)
}
