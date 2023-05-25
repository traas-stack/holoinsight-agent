/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package engine

import (
	"context"
	"github.com/docker/docker/api/types"
	dockersdk "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/system"
	"github.com/pkg/errors"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"os"
	"path/filepath"
)

func copyToContainerByDockerAPI(docker *dockersdk.Client, ctx context.Context, c *cri.Container, srcPath, dstPath string) error {
	cid := c.Id

	// ref: https://github.com/docker/cli/blob/master/cli/command/container/cp.go

	// Prepare destination copy info by stat-ing the container path.
	dstInfo := archive.CopyInfo{Path: dstPath}
	dstStat, err := docker.ContainerStatPath(ctx, cid, dstPath)

	// If the destination is a symbolic link, we should evaluate it.
	if err == nil && dstStat.Mode&os.ModeSymlink != 0 {
		linkTarget := dstStat.LinkTarget
		if !system.IsAbs(linkTarget) {
			// Join with the parent directory.
			dstParent, _ := archive.SplitPathDirEntry(dstPath)
			linkTarget = filepath.Join(dstParent, linkTarget)
		}

		dstInfo.Path = linkTarget
		dstStat, err = docker.ContainerStatPath(ctx, cid, linkTarget)
	}

	// Validate the destination path
	if err := ValidateOutputPathFileMode(dstStat.Mode); err != nil {
		return errors.Wrapf(err, `destination "%s:%s" must be a directory or a regular file`, cid, dstPath)
	}

	// Ignore any error and assume that the parent directory of the destination
	// path exists, in which case the copy may still succeed. If there is any
	// type of conflict (e.g., non-directory overwriting an existing directory
	// or vice versa) the extraction will fail. If the destination simply did
	// not exist, but the parent directory does, the extraction will still
	// succeed.
	if err == nil {
		dstInfo.Exists, dstInfo.IsDir = true, dstStat.Mode.IsDir()
	}

	var (
		resolvedDstPath string
	)

	// Prepare source copy info.
	srcInfo, err := archive.CopyInfoSourcePath(srcPath, false)
	if err != nil {
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		return err
	}
	defer srcArchive.Close()

	// With the stat info about the local source as well as the
	// destination, we have enough information to know whether we need to
	// alter the archive that we upload so that when the server extracts
	// it to the specified directory in the container we get the desired
	// copy behavior.

	// See comments in the implementation of `archive.PrepareArchiveCopy`
	// for exactly what goes into deciding how and whether the source
	// archive needs to be altered for the correct copy behavior when it is
	// extracted. This function also infers from the source and destination
	// info which directory to extract to, which may be the parent of the
	// destination that the user specified.
	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		return err
	}
	defer preparedArchive.Close()

	resolvedDstPath = dstDir
	content := preparedArchive

	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
		CopyUIDGID:                false,
	}

	return docker.CopyToContainer(ctx, cid, resolvedDstPath, content, options)
}

func copyFromContainerByDockerAPI(client *dockersdk.Client, ctx context.Context, c *cri.Container, src, dst string) error {
	content, stat, err := client.CopyFromContainer(ctx, c.Id, src)
	if err != nil {
		return err
	}
	defer content.Close()

	srcInfo := archive.CopyInfo{
		Path:   src,
		Exists: true,
		IsDir:  stat.Mode.IsDir(),
	}

	return archive.CopyTo(content, srcInfo, dst)
}

func ValidateOutputPathFileMode(fileMode os.FileMode) error {
	switch {
	case fileMode&os.ModeDevice != 0:
		return errors.New("got a device")
	case fileMode&os.ModeIrregular != 0:
		return errors.New("got an irregular file")
	}
	return nil
}
