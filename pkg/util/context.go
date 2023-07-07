/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"golang.org/x/net/context"
	"time"
)

func IsContextNotDone(ctx context.Context) bool {
	return !IsContextDone(ctx)
}

func IsContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func SubContextTimeout(ctx context.Context, timeout time.Duration, callback func(context.Context)) {
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	callback(ctx2)
}

func SubContextTimeoutE(ctx context.Context, timeout time.Duration, callback func(context.Context) error) error {
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return callback(ctx2)
}
