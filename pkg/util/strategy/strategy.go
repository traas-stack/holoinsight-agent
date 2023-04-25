/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package strategy

type RetryStrategy interface {
	GetNext() int
	Reset()
}
