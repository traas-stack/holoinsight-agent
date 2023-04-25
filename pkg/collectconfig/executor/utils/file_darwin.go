/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package utils

import (
	"os"
	"syscall"
)

func GetInode(stat os.FileInfo) uint64 {
	// darwin 也有 Ino
	return stat.Sys().(*syscall.Stat_t).Ino
}
