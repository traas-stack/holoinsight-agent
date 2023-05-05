package executor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStrReplacer(t *testing.T) {
	sr, err := parseStrReplacer("a-$1x-x${2}x,x${3}x,$4abc")
	if err != nil {
		panic(err)
	}
	str := sr.replace([]string{"_", "aaa", "bbb", "ccc"})
	assert.Equal(t, "a-aaax-xbbbx,xcccx,$4abc", str)
}
