/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	"encoding/json"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	model2 "github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input"
	_ "github.com/traas-stack/holoinsight-agent/pkg/plugin/input/all"
	"os"
)

type (
	CollectInputRequest struct {
		InputType string
		State     []byte
	}
	CollectInputResponse struct {
		Datum []*model2.DetailData
		State []byte
	}
)

func collectInputHandler(action string, resp *model.Resp) error {
	req := &CollectInputRequest{}
	err := json.NewDecoder(os.Stdin).Decode(req)
	if err != nil {
		return err
	}

	i, err := input.Parse(req.InputType, nil)
	if err != nil {
		return err
	}
	si, _ := i.(input.StatefulInput)
	if si != nil {
		si.DeserializeState(req.State)
	}

	datum, err := i.Collect(&input.CollectContext{})
	if err != nil {
		return err
	}
	inputResp := &CollectInputResponse{}
	inputResp.Datum = datum

	if si != nil {
		newState, err := si.SerializeState()
		if err != nil {
			return err
		}
		inputResp.State = newState
	}

	resp.Success = true
	resp.Data = inputResp

	return nil
}
