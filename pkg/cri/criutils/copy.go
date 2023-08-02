/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/cricore"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	context2 "golang.org/x/net/context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	tempFilePrefix               = ".temp_holoinsight_"
	tempFilePattern              = tempFilePrefix + "*"
	chunkCpOnceTimeout           = 15 * time.Second
	defaultChunkSize             = 90 * 1024
	copyToContainerByTarDisabled = "copyToContainerByTarDisabled"
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

// ReadContainerFileUsingExecCat reads container file using docker exec 'cat ...'
func ReadContainerFileUsingExecCat(ctx context.Context, i cri.Interface, c *cri.Container, path string) ([]byte, error) {
	r, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"cat", path}})
	if err != nil {
		return nil, err
	}
	return r.Stdout.Bytes(), nil
}

// CopySmallFileFromContainerToTempFile copies small file from container to temp file.
func CopySmallFileFromContainerToTempFile(ctx context.Context, i cri.Interface, c *cri.Container, path string) (string, func() error, error) {
	if s, f, err := CopyFromContainerToTempFile(ctx, i, c, path); err == nil {
		return s, f, err
	}

	// fallback to 'exec cat'
	if b, err := ReadContainerFileUsingExecCat(ctx, i, c, path); err == nil {
		f, err := os.CreateTemp("", tempFilePattern)
		if err != nil {
			return "", nil, err
		}
		_, err = io.Copy(f, bytes.NewBuffer(b))
		if err != nil {
			f.Close()
			os.Remove(f.Name())
			return "", nil, err
		}
		f.Close()
		return f.Name(), func() error {
			return os.Remove(f.Name())
		}, nil
	} else {
		return "", nil, err
	}
}

// CopyFromContainerToTempFile copies file from container to local tep file.
// It is the caller's responsibility to remove the file when it is no longer needed. See os.CreateTemp.
func CopyFromContainerToTempFile(ctx context.Context, i cri.Interface, c *cri.Container, path string) (string, func() error, error) {
	f, err := os.CreateTemp("", tempFilePattern)
	if err != nil {
		return "", nil, err
	}
	f.Close()

	if err = i.CopyFromContainer(ctx, c, path, f.Name()); err != nil {
		os.Remove(f.Name())
		return "", nil, err
	}
	return f.Name(), func() error {
		return os.Remove(f.Name())
	}, nil
}

func CopyToContainerByMount(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) (retErr error) {
	if len(c.Mounts) == 0 {
		return ErrUnsupported
	}

	// mkdir -p
	if _, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"mkdir", "-p", filepath.Dir(dstPath)}}); err != nil {
		return err
	}

	// try 'mount' workaround
	// Find first available mount. Use it as a 'transfer station':
	// 1. cp file to 'transfer station'
	// 2. mv file from 'transfer station' to dstPath using 'docker exec'
	for _, mount := range c.Mounts {
		if mount.Destination == "/dev/shm" {
			continue
		}
		tempFile := tempFilePrefix + filepath.Base(dstPath)
		tempPath := filepath.Join(mount.Source, tempFile)
		if err := util.CopyFileUsingCp(ctx, srcPath, tempPath); err != nil {
			logger.Errorz("[pouch] cp error", //
				zap.String("cid", c.Id),
				zap.Error(err))
			continue
		}

		// We assume that there is a mv command in each container.
		// Some thin images do not contain mv. Such as prometheus node exporter.
		if err := ExecMv(ctx, i, c, filepath.Join(mount.Destination, tempFile), dstPath); err != nil {
			logger.Errorz("[pouch] mv error",
				zap.String("cid", c.Id),           //
				zap.String("mount", mount.Source), //
				zap.String("src", srcPath),        //
				zap.String("dst", dstPath),        //
				zap.Error(err))
			// remove temp file
			os.Remove(tempPath)
			continue
		}

		// workaround success
		logger.Criz("[digest] copy to container", //
			zap.String("cid", c.Id),           //
			zap.String("mount", mount.Source), //
			zap.String("src", srcPath),        //
			zap.String("dst", dstPath),        //
		)
		return nil
	}

	// 'docker exec' workaround. This workaround is a bit slow.
	// Technical principle: docker exec -i "sh -c 'cat >/dstPath && chmod a+x /dstPath' < /srcPath

	// But there is a bug in rund <2.7.19, it may lose some input stream leading to a broken file (not executable).

	return ErrUnsupported
}

func CopyFromContainerByMount(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) error {
	if len(c.Mounts) == 0 {
		return ErrUnsupported
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	// TODO impl workaround for pouch:
	// 1. If target file is in mounts then `docker cp` will success.
	// 2. just like 'copyToContainerForPouch': Find first available mount. Use it as a 'transfer station':
	// 2.1. cp srcFile to 'transfer station' using 'docker exec mv'
	// 2.2. mv file from 'transfer station' to agent dstFile using
	for _, mount := range c.Mounts {
		if mount.Destination == "/dev/shm" {
			continue
		}

		tempFile := tempFilePrefix + filepath.Base(srcPath)
		tempPath := filepath.Join(mount.Destination, tempFile)
		if err := ExecCp(ctx, i, c, srcPath, tempPath); err != nil {
			logger.Errorz("[pouch] cp error",
				zap.String("cid", c.Id),                //
				zap.String("mount", mount.Destination), //
				zap.String("src", srcPath),             //
				zap.String("dst", tempPath),            //
				zap.Error(err))
			continue
		}

		tempPathInHost := filepath.Join(mount.Source, tempFile)
		if err := util.CopyFileUsingCp(ctx, tempPathInHost, dstPath); err != nil {
			logger.Errorz("[pouch] cp error", //
				zap.String("cid", c.Id),
				zap.Error(err))
			os.Remove(tempPathInHost)
			continue
		}

		// workaround success
		logger.Criz("[digest] copy from container", //
			zap.String("cid", c.Id),           //
			zap.String("mount", mount.Source), //
			zap.String("src", srcPath),        //
			zap.String("dst", dstPath),        //
		)
		return nil
	}

	return ErrUnsupported
}

// CopyToContainerByTar copies file to container using tar
func CopyToContainerByTar(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) (retErr error) {
	if l, ok := c.Attributes.Load(copyToContainerByTarDisabled); ok && l.(bool) {
		return ErrDisabled
	}

	defer func() {
		if retErr != nil && strings.Contains(retErr.Error(), "short read") {
			c.Attributes.Store(copyToContainerByTarDisabled, true)
		}
	}()

	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	preader, pwriter := io.Pipe()
	defer func() {
		pwriter.Close()
		preader.Close()
	}()

	writeErrCh := make(chan error, 1)
	go func() {
		tw := tar.NewWriter(pwriter)
		defer func() {
			tw.Close()
			pwriter.Close()
		}()

		header, err := tar.FileInfoHeader(stat, "")
		if err != nil {
			writeErrCh <- err
			return
		}
		header.Name = filepath.Base(dstPath)
		if err := tw.WriteHeader(header); err != nil {
			writeErrCh <- err
			return
		}
		if _, err := io.Copy(tw, file); err != nil {
			writeErrCh <- err
		}
	}()

	execErrCh := make(chan error, 1)
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		_, err = i.Exec(execCtx, c, cri.ExecRequest{
			Cmd:        []string{"tar", "-xmf", "-", "-C", filepath.Dir(dstPath)},
			Env:        nil,
			WorkingDir: "",
			Input:      preader,
			User:       "root",
		})
		execErrCh <- err
	}()

	select {
	case err := <-writeErrCh:
		return err
	case err := <-execErrCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CopyFromContainerByTar copies file from container using tar
func CopyFromContainerByTar(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	r, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"tar", "-cf", "-", srcPath}})
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(r.Stdout)

	_, err = tarReader.Next()
	if err != nil {
		return err
	}

	outFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	if _, err := io.Copy(outFile, tarReader); err != nil {
		return err
	}
	return nil
}

func readFileToChunks(path string) ([][]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	size := stat.Size()
	chunks := make([][]byte, 0, size/defaultChunkSize+1)

	for off := int64(0); off < size; {
		end := off + defaultChunkSize
		if end > size {
			end = size
		}
		buf := make([]byte, end-off)
		n, err := file.Read(buf)
		buf = buf[:n]
		if n > 0 {
			chunks = append(chunks, buf)
			off += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return chunks, nil
}

func CopyToContainerByChunk(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) error {
	// splits input stream of srcPath into multi chunks
	// for every chunk: put it on env (base64)
	// then exec: sh -c 'echo -n $ARG | base64 -d | dd oflag=append status=none conv=notrunc of=/tmp/1'
	// then exec: chmod a+x /tmp/1
	tempPath := dstPath + ".temp"
	if _, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"mkdir", "-p", filepath.Dir(dstPath)}}); err != nil {
		return err
	}

	stat, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	chunks, err := readFileToChunks(srcPath)
	if err != nil {
		return err
	}

	// When transfer HoloInsight helper (about 18MB) with docker+rund
	// batch size -> transfer cost:
	// 1 -> 7.4s
	// 2 -> 5.5s
	// 3 -> 4.9s
	// 4 -> 4.8s
	// 5 -> 4.4s
	var batch [][]byte
	accSize := 0
	for j, chunk := range chunks {
		batch = append(batch, chunk)
		accSize += len(chunk)
		if j+1 == len(chunks) || len(batch) >= 5 {
			env := []string{
				fmt.Sprintf("DST=%s", tempPath),
				fmt.Sprintf("CHUNK_COUNT=%d", len(batch)),
			}
			for k := range batch {
				b64 := base64.StdEncoding.EncodeToString(batch[k])
				env = append(env, fmt.Sprintf("CHUNK_ARG_%d=%s", k+1, b64))
			}

			_, err = SubContextTimeoutExec(ctx, chunkCpOnceTimeout, func(ctx2 context2.Context) (cri.ExecResult, error) {
				return i.Exec(ctx2, c, cri.ExecRequest{
					Cmd: []string{"sh", "-c", "set -e; for i in `seq $CHUNK_COUNT`; do\n  printenv CHUNK_ARG_${i} | base64 -d >> $DST \n done\n"},
					Env: env,
				})
			})

			if err != nil {
				logger.Criz("[digest] copy chunk error", zap.String("cid", c.ShortContainerID()), zap.Int("accSize", accSize), zap.Error(err))
				return err
			}

			batch = nil
			accSize = 0
		}
	}

	// mv ${dstPath}.temp ${dstPath}
	if err := ExecMv(ctx, i, c, tempPath, dstPath); err != nil {
		return err
	}

	// example: chmod 0755 dstPath
	if err := ExecChmod(ctx, i, c, dstPath, stat.Mode().Perm()); err != nil {
		return err
	}

	return nil
}

func CopyToContainer(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) (string, error) {
	var errs []error

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	if c.Runtime == cri.Runc {
		if err := cricore.CopyToContainerForRunC(ctx, c, srcPath, dstPath); err == nil {
			return cri.Runc, nil
		} else {
			errs = append(errs, err)
		}
	}

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	if err := CopyToContainerByMount(ctx, i, c, srcPath, dstPath); err == nil {
		return "mount", nil
	} else {
		errs = append(errs, err)
	}

	if i.Engine().Supports(cri.ContainerEngineFeatureCopy) {
		if util.IsContextDone(ctx) {
			return "unknown", ctx.Err()
		}

		if err := i.Engine().CopyToContainer(ctx, c, srcPath, dstPath); err == nil {
			return "engine", nil
		} else {
			errs = append(errs, err)
		}
	}

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	if err := CopyToContainerByTar(ctx, i, c, srcPath, dstPath); err == nil {
		return "tar", nil
	} else {
		errs = append(errs, err)
	}

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	if err := CopyToContainerByChunk(ctx, i, c, srcPath, dstPath); err == nil {
		return "chunk", nil
	} else {
		errs = append(errs, err)
	}

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	return "unknown", mergeError("copy to container error: ", errs)
}

func CopyFromContainer(ctx context.Context, i cri.Interface, c *cri.Container, srcPath, dstPath string) (string, error) {
	var errs []error

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	if c.Runtime == cri.Runc {
		if err := cricore.CopyFromContainerForRunC(ctx, c, srcPath, dstPath); err == nil {
			return cri.Runc, nil
		}
	}

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	if err := CopyFromContainerByMount(ctx, i, c, srcPath, dstPath); err == nil {
		return "mount", nil
	}

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	if i.Engine().Supports(cri.ContainerEngineFeatureCopy) {
		if util.IsContextDone(ctx) {
			return "unknown", ctx.Err()
		}
		if err := i.Engine().CopyFromContainer(ctx, c, srcPath, dstPath); err == nil {
			return "engine", nil
		}
	}

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	if err := CopyFromContainerByTar(ctx, i, c, srcPath, dstPath); err == nil {
		return "tar", nil
	}

	if util.IsContextDone(ctx) {
		return "unknown", ctx.Err()
	}

	return "unknown", mergeError("copy from container error: ", errs)
}

func mergeError(prefix string, errs []error) error {
	b := strings.Builder{}
	b.WriteString(prefix)
	for i, err := range errs {
		if err == ErrUnsupported {
			continue
		}
		b.WriteString(err.Error())
		if i+1 < len(errs) {
			b.WriteByte(' ')
		}
	}
	return errors.New(b.String())
}
