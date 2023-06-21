/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

import (
	"context"
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"strings"
)

// Md5sum runs 'md5sum' inside the container and extracts the md5 string
func Md5sum(ctx context.Context, i cri.Interface, c *cri.Container, path string) (string, error) {
	r, err := i.Exec(ctx, c, cri.ExecRequest{Cmd: []string{"md5sum", path}})
	if err != nil {
		return "", err
	}
	// stdout: ${md5}  ${filename}
	stdout := r.Stdout.String()
	ss := strings.SplitN(stdout, " ", 2)
	if len(ss) > 0 {
		return ss[0], nil
	}
	return "", errors.New("invalid stdout:" + stdout)
}
