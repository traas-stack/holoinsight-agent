/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package listfiles

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	fmt.Println(filepath.Clean("/saas/g/"))
	fmt.Println(filepath.Dir("/saas/g/"))
}
