package traffic

import "github.com/traas-stack/holoinsight-agent/pkg/plugin/input"

func init() {
	input.Register("traffic", func(config input.Config) (input.Input, error) {
		return &trafficInput{}, nil
	})
}
