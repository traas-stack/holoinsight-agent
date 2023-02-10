package handlers

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"os"
)

func catHandler(action string, resp *model.Resp) error {
	if len(os.Args) != 3 {
		return errors.New("miss args")
	}
	file := os.Args[2]
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	resp.Data = content
	resp.Success = true
	return nil
}
