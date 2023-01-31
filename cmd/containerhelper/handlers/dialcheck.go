package handlers

import (
	"github.com/TRaaSStack/holoinsight-agent/cmd/containerhelper/model"
	"encoding/json"
	"net"
	"os"
	"time"
)

type (
	DialCheckRequest struct {
		Network string        `json:"network"`
		Addr    string        `json:"addr"`
		Timeout time.Duration `json:"timeout"`
	}
	DialCheckResponse struct {
		Cost time.Duration `json:"cost"`
	}
)

func dialcheckHandler(action string, resp *model.Resp) error {
	req := &DialCheckRequest{}
	if err := json.NewDecoder(os.Stdin).Decode(req); err != nil {
		return err
	}

	if req.Network == "" {
		req.Network = "tcp"
	}
	if req.Timeout == 0 {
		req.Timeout = 3 * time.Second
	}

	bizResp := &DialCheckResponse{}
	resp.Data = bizResp

	dialBegin := time.Now()
	conn, err := net.DialTimeout(req.Network, req.Addr, req.Timeout)
	bizResp.Cost = time.Now().Sub(dialBegin)
	if err != nil {
		return err
	} else {
		conn.Close()
		resp.Success = true
	}
	return nil
}
