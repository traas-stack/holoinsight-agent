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
