/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package previewlog

import (
	"bytes"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"os"
)

const (
	maxBytes = 1024 * 1024
)

func PreviewFile(req *pb.PreviewFileRequest, resp *pb.PreviewFileResponse) error {
	content, err := previewFile0(req)
	if err != nil {
		return err
	}
	resp.Content = content
	return nil
}

func previewFile0(req *pb.PreviewFileRequest) ([]string, error) {
	file, err := os.Open(req.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
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
		return nil, err
	}
	dst = dst[:n]
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
				content = append(content, string(dst[offset+1:]))
			}
			dst = dst[:offset]
		}
	}
	util.ReverseStringSlice(content)

	return content, nil
}
