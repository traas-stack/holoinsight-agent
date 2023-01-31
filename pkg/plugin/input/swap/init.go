package mem

import "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input"

func init() {
	input.Register("swap", func(config input.Config) (input.Input, error) {
		return &swapInput{}, nil
	})
}
