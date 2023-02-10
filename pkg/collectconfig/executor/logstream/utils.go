package logstream

import (
	"bytes"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"os"
)

const (
	// default look back 64KB
	defaultLookback = 64 * 1024
)

// ReadLastLines reads at-most last n lines of path
func ReadLastLines(path string, required int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := info.Size()
	if fileSize == 0 {
		return nil, nil
	}

	buf := make([]byte, defaultLookback)
	var lines []string
	readAt := fileSize - defaultLookback
	if readAt < 0 {
		readAt = 0
	}
	// discardFirst is used trim last blank
	discardFirst := true

loop1:
	for {
		if readAt+defaultLookback > fileSize {
			buf = buf[:fileSize-readAt]
		} else {
			buf = buf[:defaultLookback]
		}
		n, err := file.ReadAt(buf, readAt)
		if err != nil {
			return nil, err
		}
		buf = buf[:n]

		nextReadAt := readAt
		for {
			// find last '\n' in buf
			index := bytes.LastIndexByte(buf, '\n')
			if index < 0 {
				// means that there is no line-breaker in last piece of buf
				// maybe there is a very long line, but we have to break loop to protected myself
				if readAt == 0 {
					lines = append(lines, string(buf))
				}
				break
			}

			// trim \r too
			if index > 0 && buf[index-1] == '\r' {
				index -= 1
			}

			nextReadAt = readAt + int64(index-defaultLookback)

			if discardFirst {
				discardFirst = false
			} else {
				lines = append(lines, string(buf[index+1:]))
			}

			// TODO 0 check
			if index > 0 {
				buf = buf[:index]
				if len(lines) >= required {
					break loop1
				}
			} else {
				break
			}
		}
		readAt = nextReadAt
		if readAt < 0 {
			break
		}
	}

	util.ReverseStringSlice(lines)
	return lines, nil
}
