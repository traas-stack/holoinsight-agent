/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"fmt"
	"testing"
)

func TestSlice(t *testing.T) {
	var a []string = nil
	fmt.Println(a)
	fmt.Println(a[:0])
	fmt.Println(a[:])
}
