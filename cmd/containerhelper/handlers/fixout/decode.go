/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package fixout

import (
	"encoding/binary"
	"io"
)

func Decode(hr *io.PipeReader, stdoutW io.WriteCloser, stderrW io.WriteCloser) error {
	var err error
	activeFdCount := 2
	for activeFdCount > 0 {
		var fd byte
		var size int16
		if err = binary.Read(hr, binary.LittleEndian, &fd); err != nil {
			break
		}
		if err = binary.Read(hr, binary.LittleEndian, &size); err != nil {
			break
		}
		if size == -1 {
			activeFdCount--
			switch fd {
			case StdoutFd:
				stdoutW.Close()
			case StderrFd:
				stderrW.Close()
			}
			continue
		}
		switch fd {
		case StdoutFd:
			_, err = io.CopyN(stdoutW, hr, int64(size))
		case StderrFd:
			_, err = io.CopyN(stderrW, hr, int64(size))
		}
		if err != nil {
			return err
		}
	}
	return nil
}
