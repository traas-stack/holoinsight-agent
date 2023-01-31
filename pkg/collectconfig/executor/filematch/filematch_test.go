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
