/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cri

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxDeep = 3
)

var (
	tooManySymbolLinks = errors.New("too many symbol links")
)

// TransferToHostPathForContainer transfer container path into host path
func TransferToHostPathForContainer(c *Container, ctrPath string, respectSymbol bool) (string, error) {
	return transferToHostPath0(c.MergedDir, c.Mounts, ctrPath, respectSymbol, 0)
}

// TransferToHostPath transfer container path into host path
func TransferToHostPath(mergedDir string, mounts []*MountPoint, ctrPath string, respectSymbol bool) (string, error) {
	return transferToHostPath0(mergedDir, mounts, ctrPath, respectSymbol, 0)
}

// transferToHostPath0 transfer container path into host path. This method takes into account the case of symbolic links.
func transferToHostPath0(mergedDir string, mounts []*MountPoint, ctrPath string, respectSymbol bool, deep int) (string, error) {
	if deep > maxDeep {
		return "", tooManySymbolLinks
	}

	hostPath, ok := transferToHostPath1(mergedDir, mounts, ctrPath)
	if !ok {
		return "", errors.New(ctrPath + " is not in any mount volume")
	}
	if respectSymbol {
		s, err := os.Lstat(hostPath)
		if err != nil {
			if os.IsNotExist(err) {
				return hostPath, nil
			}
			return "", err
		}
		// 处理软链接case
		if (s.Mode() & os.ModeSymlink) == os.ModeSymlink {
			r, err := os.Readlink(hostPath)
			if err != nil {
				return "", err
			}
			return transferToHostPath0(mergedDir, mounts, r, respectSymbol, deep+1)
		}
	}
	return hostPath, nil
}

// transferToHostPath1 transfer container path into host path
func transferToHostPath1(mergedDir string, mounts []*MountPoint, ctrPath string) (string, bool) {
	// clean ctrPath first to prevent from accessing to any directory using path like '../../../../home/admin/.ssh/secret' !!!
	// '/a/b/../../..' will be cleaned to '/'
	ctrPath = filepath.Clean(ctrPath)

	for _, mount := range mounts {
		if strings.HasPrefix(ctrPath, mount.Destination) {
			ld := len(mount.Destination)
			if len(ctrPath) == ld {
				return mount.Source, true
			}

			if ctrPath[len(mount.Destination)] == '/' {
				return filepath.Join(mount.Source, ctrPath[len(mount.Destination):]), true
			}
		}
	}

	if mergedDir == "" {
		return "", false
	}

	return filepath.Join(mergedDir, ctrPath), true
}
