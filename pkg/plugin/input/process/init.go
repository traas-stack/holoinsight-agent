package process

import "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input"

func init() {
	input.Register("process", func(config input.Config) (input.Input, error) {
		return &processInput{}, nil
	})
}
