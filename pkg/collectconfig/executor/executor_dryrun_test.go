package executor

import (
	"encoding/json"
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/appconfig"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collectconfig/executor/dryrun/event"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collecttask"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"io"
	"os"
	"testing"
)

func TestDryRun(t *testing.T) {
	basename := "simple"
	// log pipeline
	appconfig.SetDev(true)
	logger.DebugEnabled = true

	logger.SetupZapLogger()

	go generateLogs(basename)
	f, err := os.Open("examples/" + basename + "/1.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	configContentBytes, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}

	ct := &collecttask.CollectTask{
		Key:     "TEST",
		Version: "TEST",
		Config: &collecttask.CollectConfig{
			Key:     "",
			Type:    "SQLTASK",
			Version: "",
			Content: configContentBytes,
		},
		Target: &collecttask.CollectTarget{},
	}

	e, err := NewDryRunExecutor(&DryRunRequest{
		Task: ct,
		Input: &Input{
			Type:  "read",
			Plain: &Input_Plain{Lines: []string{"2022-12-29 16:38:01 INFO a=1 b=4x flag=true"}},
			Read:  &Input_Read{MaxLines: 10},
		},
	})
	if err != nil {
		panic(err)
	}
	resp := e.Run()
	fmt.Println(resp.Event)

	dfsPrintEvent(resp.Event, "")

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(resp.Event)
}

func dfsPrintEvent(event *event.Event, indent string) {
	str := util.ToJsonString(event.Params)
	if str == "null" {
		str = "{}"
	}
	fmt.Printf("%s%s %v\n", indent, event.Title, str)

	for _, sub := range event.Messages {
		fmt.Printf("  %s [%s] %s\n", indent, sub.Level, sub.Content)
	}
	nextIndent := indent + " "
	for _, childEvent := range event.Children {
		dfsPrintEvent(childEvent, nextIndent)
	}
}
