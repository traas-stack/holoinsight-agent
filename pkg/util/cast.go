/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

// BoolToInt64 converts bool to int64
func BoolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
