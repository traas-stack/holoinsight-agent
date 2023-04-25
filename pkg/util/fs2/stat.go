/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package fs2

import (
	"errors"
	"os"
	"path/filepath"
)

func Stat(root, path string, maxDepth int) (os.FileInfo, string, error) {
	for i := 0; i <= maxDepth; i++ {
		join := filepath.Join(root, path)
		st, err := os.Lstat(join)
		if err != nil {
			return nil, "", err
		}

		if st.Mode()&os.ModeSymlink == 0 {
			return st, path, nil
		}

		link, err := os.Readlink(join)
		if err != nil {
			return nil, "", err
		}

		path = link
	}
	return nil, "", errors.New("reach max depth")
}

func Stat2(path string, maxDepth int) (os.FileInfo, string, error) {
	for i := 0; i <= maxDepth; i++ {
		st, err := os.Lstat(path)
		if err != nil {
			return nil, "", err
		}

		if st.Mode()&os.ModeSymlink == 0 {
			return st, path, nil
		}

		link, err := os.Readlink(path)
		if err != nil {
			return nil, "", err
		}

		path = link
	}
	return nil, "", errors.New("reach max depth")
}
