/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package previewlog

import (
	"bytes"
	"github.com/saintfish/chardet"
	"github.com/stretchr/testify/assert"
	"golang.org/x/text/encoding/simplifiedchinese"
	"os/exec"
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	cmd := exec.Command("iconv", "-f", "UTF-8", "-t", "GB18030")
	cmd.Stdin = strings.NewReader("我是中文哦哦哦")
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	cmd.Run()

	assert.NotEqual(t, "我是中文哦哦哦", stdout.String())

	containsGb18030 := false
	results, err := chardet.NewTextDetector().DetectAll(stdout.Bytes())
	assert.NoError(t, err)
	for _, result := range results {
		if result.Charset == "GB-18030" {
			containsGb18030 = true
			break
		}
	}
	assert.True(t, containsGb18030)

	d, err := simplifiedchinese.GB18030.NewDecoder().Bytes(stdout.Bytes())
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "我是中文哦哦哦", string(d))
}
