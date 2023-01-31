package dockerutils

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/core"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"os"
	"path/filepath"
	"strings"
)

var defaultDocketSock = "/var/run/docker.sock,/var/run/pouchd.sock"
var dockerSocks = util.TransformStringSlice(
	strings.Split(util.GetEnvOrDefault("DOCKER_SOCK", defaultDocketSock), ","),
	strings.TrimSpace,
)

func DetectSock() (string, bool) {
	for _, path := range dockerSocks {
		hostPath := filepath.Join(core.GetHostfs(), path)
		if _, err := os.Stat(hostPath); err == nil {
			// unixsocket 是什么类型?
			return hostPath, true
		}
	}
	return "", false
}
