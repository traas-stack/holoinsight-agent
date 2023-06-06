/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"bytes"
	"encoding/gob"
)

func GobEncode(obj interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := gob.NewEncoder(buf).Encode(obj)
	return buf.Bytes(), err
}

func GobDecode(b []byte, obj interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(b)).Decode(obj)
}
