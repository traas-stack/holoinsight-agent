package executor

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestName(t *testing.T) {
	f, err := os.Open("demo1.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	sql := &collectconfig.SQLTask{}
	json.Unmarshal(b, sql)
	fmt.Printf("%+v\n", sql)
}
