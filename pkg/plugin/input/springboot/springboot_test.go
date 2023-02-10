package springboot

import (
	"encoding/json"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/accumulator"
	telegraf2 "github.com/traas-stack/holoinsight-agent/pkg/telegraf"
	"os"
	"testing"
)

func TestSpringBoot(t *testing.T) {
	b := &SpringBoot{BaseUrl: "http://11.166.169.151:8080/actuator"}
	m := &accumulator.Memory{}
	err := b.Gather(telegraf2.Adapt(m, "_"))
	fmt.Println(err)
	json.NewEncoder(os.Stdout).Encode(m.Metrics)
}
