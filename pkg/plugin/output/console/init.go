package console

import "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/output"

func init() {
	output.Register("console", func(config output.Config) (output.Output, error) {
		return NewConsoleOutput()
	})
}
