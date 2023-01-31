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
	invalidMergedDir   = errors.New("invalid MergeDir")
)

func TransferToHostPath0(c *Container, ctrPath string, respectSymbol bool) (string, error) {
	return transferToHostPath0(c.MergedDir, c.Mounts, ctrPath, respectSymbol, 0)
}

func TransferToHostPath(mergedDir string, mounts []*MountPoint, ctrPath string, respectSymbol bool) (string, error) {
	return transferToHostPath0(mergedDir, mounts, ctrPath, respectSymbol, 0)
}

func transferToHostPath0(mergedDir string, mounts []*MountPoint, ctrPath string, respectSymbol bool, deep int) (string, error) {
	if deep > maxDeep {
		return "", tooManySymbolLinks
	}

	hostPath, err := transferToHostPath1(mergedDir, mounts, ctrPath)
	if err != nil {
		return "", err
	}
	if respectSymbol {
		s, err := os.Lstat(hostPath)
		if err != nil {
			// 这里只是要转换路径, 不用确保文件可读
			if os.IsNotExist(err) || os.IsPermission(err) {
				return hostPath, nil
			}
			// 其他错误 啥?
			return hostPath, nil
		}
		// 处理软链接case
		if (s.Mode() & os.ModeSymlink) == os.ModeSymlink {
			r, err := os.Readlink(hostPath)
			if err != nil {
				return "", err
			}
			// 递归解释
			return transferToHostPath0(mergedDir, mounts, r, respectSymbol, deep+1)
		}
	}
	return hostPath, nil
}

func transferToHostPath1(mergedDir string, mounts []*MountPoint, ctrPath string) (string, error) {
	for _, mount := range mounts {
		if strings.HasPrefix(ctrPath, mount.Destination) {
			ld := len(mount.Destination)
			if len(ctrPath) == ld {
				return filepath.Join(mount.Source), nil
			}

			if ctrPath[len(mount.Destination)] == '/' {
				return filepath.Join(mount.Source, ctrPath[len(mount.Destination):]), nil
			}
		}
	}

	// TODO 可以 exec 进去执行 realpath (如果realpath不存在我们可以自带), 主要是准确
	// TODO 可以 nsenter 进去执行 realpath (如果realpath不存在我们可以自带) (比前者更快)

	if mergedDir == "" {
		return "", invalidMergedDir
	}

	return filepath.Join(mergedDir, ctrPath), nil
}

func Readlink(root, path string) (string, error) {
	link, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, link), nil
}
