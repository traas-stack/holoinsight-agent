package load

import "github.com/traas-stack/holoinsight-agent/pkg/plugin/input"

func init() {
	input.Register("load", func(config input.Config) (input.Input, error) {
		return &loadInput{}, nil
	})
}
