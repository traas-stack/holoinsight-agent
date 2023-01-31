package logstream

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	lines, err := ReadLastLines("/Users/xzchaoo/logs/a.log", 13)
	if err != nil {
		panic(err)
	}
	for _, line := range lines {
		fmt.Println(line, len(line))
	}
}
