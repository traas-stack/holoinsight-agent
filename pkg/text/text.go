/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package text

import (
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	UTF8 = "UTF-8"
)

var (
	expectedCharsets = []string{UTF8, "GB-18030"}
	decoderMap       = make(map[string]encoding.Encoding)
)

func init() {
	decoderMap["GB-18030"] = simplifiedchinese.GB18030
	// alias
	decoderMap["GB18030"] = simplifiedchinese.GB18030
	decoderMap["GBK"] = simplifiedchinese.GB18030
	decoderMap["GB2312"] = simplifiedchinese.GB18030
}

// DetectCharset detects charset from bytes
func DetectCharset(bs []byte) string {
	if charsetResults, err := chardet.NewTextDetector().DetectAll(bs); err == nil {
		for _, expected := range expectedCharsets {
			for _, result := range charsetResults {
				if result.Charset == expected {
					return result.Charset
				}
			}
		}
	}

	return UTF8
}

func GetEncoding(charset string) encoding.Encoding {
	return decoderMap[charset]
}
