/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cri

import (
	"errors"
	"github.com/spf13/cast"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
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
	if !filepath.IsAbs(ctrPath) {
		return "", errors.New("container path must be a abs path")
	}
	if c.Runtime == Runc && c.MergedDir == "" && c.State.Pid > 0 {
		if ctrPath == "/" {
			return "", errors.New("container path '/' is disallowed")
		}
		// Notice: /hostfs/proc/${pid}/root is a link to '/' .
		// When you cd to it using bash, its content belongs to the target container. That's ok!
		// But if you open this dir using Golang, it will really link to '/' of agent container!!!
		return filepath.Join(core.GetHostfs(), "proc", cast.ToString(c.State.Pid), "root", ctrPath), nil
	} else {
		return transferToHostPath0(c.MergedDir, c.Mounts, ctrPath, respectSymbol, 0)
	}
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
