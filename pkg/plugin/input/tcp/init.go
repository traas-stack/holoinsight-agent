package tcp

import "github.com/traas-stack/holoinsight-agent/pkg/plugin/input"

func init() {
	input.Register("tcp", func(config input.Config) (input.Input, error) {
		return &tcpInput{}, nil
	})
}
