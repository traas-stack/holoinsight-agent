/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package api

func BoolToFloat64(b bool) float64 {
	if b {
		return 1
	} else {
		return 0
	}
}
