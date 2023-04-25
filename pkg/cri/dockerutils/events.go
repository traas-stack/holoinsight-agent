/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package dockerutils

import "strings"

// action 形如 'dir' 'exec_create: bash -c sc stop app', 对于后者我们提取出 exec_create
func ExtractEventAction(rawAction string) string {
	if strings.Contains(rawAction, ":") {
		rawAction = rawAction[:strings.Index(rawAction, ":")]
	}
	return rawAction
}
