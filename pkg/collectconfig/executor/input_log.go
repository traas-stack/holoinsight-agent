/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

type (
	LogInput interface {
		Start()
		Stop()
		Pull(*PullRequest) (*PullResponse, error)
		Finished() bool
	}
)
