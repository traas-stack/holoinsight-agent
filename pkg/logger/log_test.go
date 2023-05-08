/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logger

import "testing"

func TestLog(t *testing.T) {
	Infof("AAAA")
	Errorf("BBBB")
}
