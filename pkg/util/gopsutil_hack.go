package util

import (
	_ "unsafe"
)

// 该error在internal包里, 默认我们无法引用, 通过 go:linkename 的方式将其暴露出来
//go:linkname errNotImplementedError github.com/shirou/gopsutil/v3/internal/common.ErrNotImplementedError
var errNotImplementedError error

func IsNotImplemented(err error) bool {
	return err == errNotImplementedError
}
