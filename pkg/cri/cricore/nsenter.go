//go:build !linux

/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cricore

import (
	"errors"
)

func NsEnterAndRunCodes(nsFile string, callback func()) error {
	return errors.New("unsupported")
}
