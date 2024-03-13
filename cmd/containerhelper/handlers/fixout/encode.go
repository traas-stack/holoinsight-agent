/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package fixout

import (
	"encoding/binary"
	"io"
	"os"
	"sync"
)

var (
	writeLock sync.Mutex
)

func write(fd int, payload []byte) error {
	writeLock.Lock()
	defer writeLock.Unlock()

	// fd 1 byte
	// size 2 bytes
	// payload <size> bytes (optional)
	if err := binary.Write(os.Stdout, binary.LittleEndian, byte(fd)); err != nil {
		return err
	}
	if err := binary.Write(os.Stdout, binary.LittleEndian, int16(len(payload))); err != nil {
		return err
	}
	return binary.Write(os.Stdout, binary.LittleEndian, payload)
}

func writeClose(fd int) error {
	writeLock.Lock()
	defer writeLock.Unlock()

	// fd 1 byte
	// -1 2 bytes (const)
	if err := binary.Write(os.Stdout, binary.LittleEndian, byte(fd)); err != nil {
		return err
	}
	return binary.Write(os.Stdout, binary.LittleEndian, int16(-1))
}

// copyStream reads bytes from in, and encodes bytes into os.Stdout
func CopyStream(fd int, in io.Reader, errChan chan error) {
	buf := make([]byte, bufSize)
	for {
		n, err := in.Read(buf)
		var err2 error
		if n > 0 {
			err2 = write(fd, buf[:n])
		}
		if err == io.EOF {
			if err2 == nil {
				err2 = writeClose(fd)
			}
		}
		if err == nil {
			err = err2
		}
		if err != nil {
			errChan <- err
			if err != io.EOF {
				io.Copy(io.Discard, in)
			}
			break
		}
	}
}
