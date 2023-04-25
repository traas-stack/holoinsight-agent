/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	o := "/home/admin/logs/foo/{time:yyyy}/{time:MM}/{time:dd}/{time:HH}/foo.log"
	f := newFilePathFormatter()
	if f.ContainsFormat(o) {
		fmt.Println(f.Format(o))
	} else {
		fmt.Println(o)
	}
}
