/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/traas-stack/holoinsight-agent/cmd/containerhelper/handlers"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"os"
	"time"
)

// This is a helper binary.
// It will be copied to container.
// It will be called by docker exec API to detect certain information inside the container.
func main() {
	// Under normal circumstances, the helper program should not be executed for a long time. If the time is exceeded, it will be forced to exit.
	time.AfterFunc(3*time.Minute, func() {
		os.Exit(97)
	})
	var resp = &model.Resp{}

	defer func() {
		if os.Getenv("NO_JSON_OUTPUT") != "true" {
			bs, _ := json.Marshal(resp)
			os.Stdout.Write(bs)
		}
		// Encoder.Encode will append a newline char
		// json.NewEncoder(os.Stdout).Encode(resp)
	}()

	if len(os.Args) == 1 {
		resp.Message = "miss action"
		return
	}

	action := os.Args[1]
	handler, ok := model.GetHandler(action)
	if !ok {
		resp.Message = fmt.Sprintf("unknown action %s", action)
		return
	}

	err := handler(action, resp)
	if err != nil {
		resp.Success = false
		if resp.Message == "" {
			resp.Message = err.Error()
		}
		if os.Getenv("NO_JSON_OUTPUT") == "true" {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}
