/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package model

type (
	RespBase struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	Resp struct {
		RespBase
		Data interface{} `json:"data"`
	}
	RespDataBytes struct {
		Base RespBase
		Data []byte `json:"data"`
	}
)
