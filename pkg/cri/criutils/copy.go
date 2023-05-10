/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"os"
)

// ReadContainerFile copies file from container to local temp file, reads the file content, and then remove the temp file.
func ReadContainerFile(ctx context.Context, i cri.Interface, c *cri.Container, path string) ([]byte, error) {
	tempPath, deleteFunc, err := CopyFromContainerToTempFile(ctx, i, c, path)
	if err != nil {
		return nil, err
	}
	defer deleteFunc()

	return os.ReadFile(tempPath)
}

// CopyFromContainerToTempFile copies file from container to local tep file.
// It is the caller's responsibility to remove the file when it is no longer needed. See os.CreateTemp.
func CopyFromContainerToTempFile(ctx context.Context, i cri.Interface, c *cri.Container, path string) (string, func() error, error) {
	f, err := os.CreateTemp("", "holoinsight-agent-copy-*")
	if err != nil {
		return "", nil, err
	}
	f.Close()

	if err = i.CopyFromContainer(ctx, c, path, f.Name()); err != nil {
		return "", nil, err
	}
	return f.Name(), func() error {
		return os.Remove(f.Name())
	}, nil
}
