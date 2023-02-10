package httpcheck

import (
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	input := &Input{
		Config: &Config{
			URL:          "https://www.taobao.com",
			Headers:      map[string]string{"foo": "bar"},
			Timeout:      311 * time.Second,
			Method:       http.MethodGet,
			SuccessCodes: []int{200},
			SuccessRegexps: []*regexp.Regexp{
				regexp.MustCompile("增加斑马关键数据监控"),
			},
			BodyLimit: 1024 * 1024,
		},
	}
	ma := api.NewMemoryAccumulator()
	err := input.Collect(ma)
	if err != nil {
		panic(err)
	}
	ma.PrintTo(os.Stdout)
}
