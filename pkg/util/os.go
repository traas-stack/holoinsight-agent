/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

var isLinux = runtime.GOOS == "linux"

func IsLinux() bool {
	return isLinux
}

func GetEnvOrDefault(name, defaultValue string) string {
	s := os.Getenv(name)
	if s == "" {
		s = defaultValue
	}
	return s
}

// CreateDirIfNotExists creates dir if it does not exist
func CreateDirIfNotExists(dir string, perm os.FileMode) error {
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, perm)
		}
		return err
	}
	if stat.IsDir() {
		return nil
	}
	return os.ErrExist
}

// CopyFileUsingCp copies file using cp binary (in agent container)
func CopyFileUsingCp(ctx context.Context, src, dst string) error {
	cmd := exec.CommandContext(ctx, "cp", src, dst)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return nil
	}
	return fmt.Errorf("cp error, cmd=[%s] code=[%d] stdout=[%s] stderr=[%s]", cmd.String(), cmd.ProcessState.ExitCode(), stdout.String(), stderr.String())
}
