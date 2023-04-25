/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package model

type (
	Handler func(action string, resp *Resp) error
)
