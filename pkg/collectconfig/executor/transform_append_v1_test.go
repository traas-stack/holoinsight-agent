package executor

import (
	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"testing"
)

func TestAppendFilterV1(t *testing.T) {
	f := xAppendV1Filter{conf: &collectconfig.TransformFilterAppendV1{
		Value:           "_SUFFIX",
		AppendIfMissing: false,
	}}
	v, _ := f.Filter(&LogContext{contextValue: "a"})
	assert.Equal(t, "a_SUFFIX", v)

	v, _ = f.Filter(&LogContext{contextValue: "a_SUFFIX"})
	assert.Equal(t, "a_SUFFIX_SUFFIX", v)

	f = xAppendV1Filter{conf: &collectconfig.TransformFilterAppendV1{
		Value:           "_SUFFIX",
		AppendIfMissing: true,
	}}
	v, _ = f.Filter(&LogContext{contextValue: "a"})
	assert.Equal(t, "a_SUFFIX", v)

	v, _ = f.Filter(&LogContext{contextValue: "a_SUFFIX"})
	assert.Equal(t, "a_SUFFIX", v)
}
