package nodehostname

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// 所属的物理机的 hostname
var nodeHostname string

func NodeHostname() string {
	return nodeHostname
}

func init() {
	if os.Getenv("NODE_IP") != "" {
		// in k8s
		path := filepath.Join(core.GetHostfs(), "/etc/hostname")
		b, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("get host /etc/hostname error: path=%s: %+v\n", path, err)
		} else {
			nodeHostname = strings.TrimSpace(string(b))
		}
	}
	if nodeHostname == "" {
		var err error
		nodeHostname, err = os.Hostname()
		if err != nil {
			fmt.Printf("get os hostname error: %+v\n", err)
		}
	}
	if nodeHostname == "" {
		nodeHostname = "unknown"
	}
}
