/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"encoding/json"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"io"
	"os"
	"testing"
)

func TestName(t *testing.T) {
	f, err := os.Open("demo1.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	sql := &collectconfig.SQLTask{}
	json.Unmarshal(b, sql)
	fmt.Printf("%+v\n", sql)
}
