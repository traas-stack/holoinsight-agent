package filematch

import (
	"fmt"
	"testing"
)

func TestRegexpFileMatcher(t *testing.T) {
	m, err := NewRegexpFileMatcher("/Users/xzchaoo/logs", "^/Users/xzchaoo/logs/(?P<type>[^/]+)/(?P<basename>[^/]+).log$", -1, 10)
	if err != nil {
		panic(err)
	}
	fmt.Println(m.Find())
}
