/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"bytes"
	"encoding/json"
)

// ToJsonString convert v to json string ignoring any error
func ToJsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func ToJsonBytes(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func ToJsonBuffer(v interface{}) *bytes.Buffer {
	bb := bytes.NewBuffer(nil)
	json.NewEncoder(bb).Encode(v)
	return bb
}

func ToJsonBufferE(v interface{}) (*bytes.Buffer, error) {
	bb := bytes.NewBuffer(nil)
	err := json.NewEncoder(bb).Encode(v)
	return bb, err
}
