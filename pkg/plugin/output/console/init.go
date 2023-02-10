package console

import "github.com/traas-stack/holoinsight-agent/pkg/plugin/output"

func init() {
	output.Register("console", func(config output.Config) (output.Output, error) {
		return NewConsoleOutput()
	})
}
