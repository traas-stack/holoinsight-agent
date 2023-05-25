/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cri

func ShortContainerId(cid string) string {
	if len(cid) <= 12 {
		return cid
	}
	return cid[:12]
}
