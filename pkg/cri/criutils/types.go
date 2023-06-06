/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

type (
	ProcessInfo struct {
		User         string   `json:"user"`
		Name         string   `json:"name"`
		CmdlineSlice []string `json:"cmdlineSlice"`
		Tgid         int32    `json:"tgid"`
	}
)
