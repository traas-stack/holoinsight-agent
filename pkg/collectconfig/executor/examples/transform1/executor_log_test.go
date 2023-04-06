package executor

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/testutils"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/output/console"
	"os"
	"testing"
	"time"
)

func TestExecutor_simple(t *testing.T) {
	// log pipeline
	appconfig.SetDev(true)
	logger.DebugEnabled = true

	basename := "transform1"
	go generateLogs(basename)

	pipeline, err := testutils.LoadLogPipeline("1.json")
	assert.NoError(t, err)

	go pipeline.Start()

	time.Sleep(20 * time.Second)
	pipeline.Stop()
}

func generateLogs(basename string) {
	f, err := os.Open("1.log")
	if err != nil {
		f, err = os.Open("1.demolog")
	}
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	f.Close()

	out1, err := os.OpenFile("/tmp/1.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	//out2, err := os.OpenFile("/Users/xzchaoo/logs/bbb/1.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		panic(err)
	}
	for {
		buf := &bytes.Buffer{}
		for _, line := range lines {
			fmt.Fprintf(buf, "%s%s\n", time.Now().Format("2006-01-02 15:04:05"), line[len("2006-01-02 15:04:05"):])
		}
		fmt.Fprint(out1, buf.String())
		time.Sleep(time.Second)
	}
}
