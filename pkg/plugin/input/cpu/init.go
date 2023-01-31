package cpu

import "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input"

func init() {
	input.Register("cpu", func(config input.Config) (input.Input, error) {
		return &cpuInput{}, nil
	})
}
