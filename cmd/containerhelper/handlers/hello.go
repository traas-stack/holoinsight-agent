/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
)

func helloHandler(inputType string, resp *model.Resp) error {
	resp.Data = "world"
	return nil
}
