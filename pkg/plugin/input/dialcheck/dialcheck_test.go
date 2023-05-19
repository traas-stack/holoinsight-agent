/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package dialcheck

import (
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"os"
	"testing"
	"time"
)

func TestDialCheck(t *testing.T) {
	input := &Input{
		Config: &Config{
			Network: "tcp",
			Host:    "127.0.0.1",
			Ports:   []int{22},
			Timeout: 3 * time.Second,
			Times:   3,
		},
	}

	ma := api.NewMemoryAccumulator()
	err := input.Collect(ma)
	if err != nil {
		panic(err)
	}
	ma.PrintTo(os.Stdout)
}
