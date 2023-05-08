/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/dialcheckw"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/inputproxy"
	"io"
	"os"
)

// inputProxyHandler proxies request to input which impls api2.InputExtNsEnter
func inputProxyHandler(_ string, resp *model.Resp) error {
	if len(os.Args) < 3 {
		return errors.New("miss args")
	}

	configType := os.Args[2]

	bs, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	provider := inputproxy.Get(configType)
	if provider == nil {
		return errors.New("unsupported configType " + configType)
	}

	extInput := provider()

	respBytes, err := extInput.ExecuteRequest(bs)
	if err != nil {
		return err
	}
	resp.Data = respBytes
	return nil
}
