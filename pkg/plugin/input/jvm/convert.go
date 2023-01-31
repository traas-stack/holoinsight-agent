package jvm

import "github.com/spf13/cast"

// 1k 1m 1g
func convertToBytes(s string) int64 {
	if s == "" {
		return 0
	}
	i64, err := cast.ToInt64E(s[0 : len(s)-1])
	if err != nil {
		return 0
	}

	switch s[len(s)-1] {
	case 'k':
		return i64 * 1024
	case 'm':
		return i64 * 1024 * 1024
	case 'g':
		return i64 * 1024 * 1024 * 1024
	default:
		// unknown unit
		return 0
	}
}
