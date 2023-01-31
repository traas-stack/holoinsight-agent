package main

import (
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/k8s/resources"
	"os"
)

// agent entry
func main() {
	resources.SetupCpuAndMemoryLimit()

	if err := bootstrap(); err != nil {
		fmt.Printf("bootstrap error %+v\n", err)
		os.Exit(1)
	}
}
