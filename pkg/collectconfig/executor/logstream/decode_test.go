/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logstream

import (
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"golang.org/x/text/encoding/simplifiedchinese"
	"testing"
)

var str = "我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文我是一个很长的中文"

func BenchmarkDecoder_string(b *testing.B) {
	s, err := simplifiedchinese.GB18030.NewEncoder().String(str)
	if err != nil {
		panic(err)
	}

	decoder := simplifiedchinese.GB18030.NewDecoder()
	for i := 0; i < b.N; i++ {
		_, _ = decoder.String(s)
	}
}

func BenchmarkDecoder_bytes(b *testing.B) {
	s, err := simplifiedchinese.GB18030.NewEncoder().String(str)
	if err != nil {
		panic(err)
	}

	bs := []byte(s)
	decoder := simplifiedchinese.GB18030.NewDecoder()
	for i := 0; i < b.N; i++ {
		d, _ := decoder.Bytes(bs)
		_ = util.String(d)
	}
}

// BenchmarkDecoder_bytes_2 is a bit faster than BenchmarkDecoder_bytes because it prepares a big enough init []byte.
func BenchmarkDecoder_bytes_2(b *testing.B) {
	s, err := simplifiedchinese.GB18030.NewEncoder().String(str)
	if err != nil {
		panic(err)
	}

	bs := []byte(s)
	decoder := simplifiedchinese.GB18030.NewDecoder()
	for i := 0; i < b.N; i++ {
		//transform.Bytes(decoder.Transformer, bs)
		dst := make([]byte, len(bs)*2)
		n, _, _ := decoder.Transform(dst, bs, true)

		// 30ns/op
		//_ = string(dst[:n])

		_ = util.String(dst[:n])

		//assert.Equal(t, str,decoded)
		// fmt.Println(n, len(bs))
		//fmt.Println(string(dst[:n]))
		//os.Exit(1)
	}
}
