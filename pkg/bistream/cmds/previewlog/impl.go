package previewlog

import (
	"bytes"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"os"
)

const (
	maxBytes = 1024 * 1024
)

func PreviewFile(req *pb.PreviewFileRequest, resp *pb.PreviewFileResponse) error {
	file, err := os.Open(req.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	readBytes := int64(req.MaxBytes)
	fileSize := stat.Size()

	// TODO const
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
		return err
	}
	dst = dst[:n]
	content := make([]string, 0)

	if true {
		// 倒序遍历
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
	} else {
		// 正序遍历
		firstLine := true
		for len(dst) > 0 {
			offset := bytes.IndexByte(dst, '\n')
			if offset < 0 {
				content = append(content, string(dst))
				break
			}
			if firstLine {
				firstLine = false
				// 解释一下这个算法
				if readOffset == 0 {
					content = append(content, string(dst[:offset]))
				}
			} else {
				// TODO trim \r
				content = append(content, string(dst[:offset]))
			}
			dst = dst[offset+1:]
		}
	}

	resp.Content = content
	return nil
}
