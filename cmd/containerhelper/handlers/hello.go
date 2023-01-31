package handlers

import (
	"github.com/TRaaSStack/holoinsight-agent/cmd/containerhelper/model"
)

func helloHandler(inputType string, resp *model.Resp) error {
	resp.Data = "world"
	return nil
}
