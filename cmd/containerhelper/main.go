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
)

// 这是一个辅助的bin, k8s场景下, 该bin会被复制到容器内, 然后 daemonset agent 会 使用 docker exec 调用该 bin 在容器内部执行一些命令(从而避免在daemonset agent 通过 切换namespace去访问)
// 入参 通过 args 和 stdin(内容是一个json) 传输
// 出参 通过 stdout(内容是一个json) 传输
func main() {
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
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}
}
