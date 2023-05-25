/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package main

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/bootstrap"
	"os"
)

// agent entry
func main() {
	if err := bootstrap.App.Bootstrap(); err != nil {
		fmt.Printf("bootstrap error %+v\n", err)
		os.Exit(1)
	}
}
