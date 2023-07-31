/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package previewlog

import (
	"bytes"
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/text"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"golang.org/x/text/encoding"
)

const (
	maxBytes = 1024 * 1024
	// UTF8 is default charset of Golang
)

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
	file, err := util.OpenFileReadonly(req.Path)
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

	charset := text.DetectCharset(dst)

	var decoder *encoding.Decoder
	if charset != text.UTF8 {
		if encoding2 := text.GetEncoding(charset); encoding2 == nil {
			return "", nil, errors.New("unsupported charset " + charset)
		} else {
			decoder = encoding2.NewDecoder()
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
				if charset != text.UTF8 {
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
