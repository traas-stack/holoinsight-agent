package handlers

import (
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/httpcheck"
	"io"
	"os"
)

func httpCheckHandler(action string, resp *model.Resp) error {
	configBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	input := &httpcheck.Input{}
	responseBytes, err := input.ExecuteRequest(configBytes)
	if err != nil {
		return err
	}

	resp.Success = true
	resp.Data = responseBytes

	return nil
}
