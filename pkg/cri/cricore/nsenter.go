//go:build !linux

/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cricore

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"net"
	"time"
)

func NsEnterAndRunCodes(nsFile string, callback func()) error {
	return errors.New("unsupported")
}

func NsEnterDial(c *cri.Container, network, addr string, timeout time.Duration) (net.Conn, error) {
	return nil, errors.New("unsupported")
}
