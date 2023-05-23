/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package previewlog

import (
	"bytes"
	"errors"
	"github.com/saintfish/chardet"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"os"
)

const (
	maxBytes = 1024 * 1024
	// UTF8 is default charset of Golang
	UTF8 = "UTF-8"
)

var (
	expectedCharsets = []string{UTF8, "GB-18030"}
	decoderMap       = make(map[string]func() *encoding.Decoder)
)

func init() {
	decoderMap["GB-18030"] = simplifiedchinese.GB18030.NewDecoder
}

func PreviewFile(req *pb.PreviewFileRequest, resp *pb.PreviewFileResponse) error {
	charset, content, err := previewFile0(req)
	if err != nil {
		return err
	}
	resp.Charset = charset
	resp.Content = content
	return nil
}

func previewFile0(req *pb.PreviewFileRequest) (string, []string, error) {
	file, err := os.Open(req.Path)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", nil, err
	}

	readBytes := int64(req.MaxBytes)
	fileSize := stat.Size()

	if readBytes > maxBytes {
		readBytes = maxBytes
	}

	// TODO 解释
	// 多读一个字节
	readBytes += 1

	if readBytes > fileSize {
		readBytes = fileSize
	}

	dst := make([]byte, readBytes)
	readOffset := stat.Size() - readBytes
	n, err := file.ReadAt(dst, readOffset)
	if err != nil {
		return "", nil, err
	}
	dst = dst[:n]

	charset := detectCharset(dst)
	logger.Infoz("detect charset", zap.String("charset", charset))

	var decoder *encoding.Decoder
	if charset != UTF8 {
		if decoderProvider, ok := decoderMap[charset]; !ok {
			return "", nil, errors.New("unsupported charset " + charset)
		} else {
			decoder = decoderProvider()
		}
	}

	content := make([]string, 0)

	firstLine := true
	maxLines := int(req.MaxLines)
	for len(dst) > 0 && maxLines > 0 && len(content) < maxLines {
		offset := bytes.LastIndexByte(dst, '\n')
		if offset < 0 {
			break
		} else {
			if firstLine {
				firstLine = false
			} else {
				decoded := ""
				if charset != UTF8 {
					if d, err := decoder.Bytes(dst[offset+1:]); err == nil {
						decoded = string(d)
					} else {
						decoded = string(dst[offset+1:])
					}
				} else {
					decoded = string(dst[offset+1:])
				}
				content = append(content, decoded)
			}
			dst = dst[:offset]
		}
	}
	util.ReverseStringSlice(content)

	return charset, content, nil
}

// detectCharset detects charset from bytes
func detectCharset(dst []byte) string {
	if charsetResults, err := chardet.NewTextDetector().DetectAll(dst); err == nil {
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
