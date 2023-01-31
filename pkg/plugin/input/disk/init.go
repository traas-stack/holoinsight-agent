package disk

import "github.com/TRaaSStack/holoinsight-agent/pkg/plugin/input"

func init() {
	input.Register("disk", func(config input.Config) (input.Input, error) {
		return &diskInput{}, nil
	})
}
