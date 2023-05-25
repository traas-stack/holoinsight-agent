package cricore

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"path/filepath"
)

func CopyToContainerForRunC(ctx context.Context, c *cri.Container, srcPath, dstPath string) error {
	hostPath, err := cri.TransferToHostPathForContainer(c, dstPath, true)
	if err != nil {
		return err
	}

	if err := util.CreateDirIfNotExists(filepath.Dir(hostPath), 0755); err != nil {
		return err
	}

	if err := util.CopyFileUsingCp(ctx, srcPath, hostPath); err != nil {
		return err
	}

	return nil
}

func CopyFromContainerForRunC(ctx context.Context, c *cri.Container, srcPath, dstPath string) error {
	hostPath, err := cri.TransferToHostPathForContainer(c, srcPath, true)
	if err != nil {
		return err
	}

	if err := util.CreateDirIfNotExists(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	if err := util.CopyFileUsingCp(ctx, hostPath, dstPath); err != nil {
		return err
	}

	return nil
}
