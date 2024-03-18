//go:build linux

/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cricore

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// NsEnterAndRunCodes enter target ns, and then run callback.
// Notice that this func will run callback in a new thread.
// After callback returns, the thread terminates.
// See runtime.LockOSThread()
func NsEnterAndRunCodes(nsFile string, callback func()) error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	var err2 error
	go func() {
		runtime.LockOSThread()
		// Don't call UnlockOSThread. This thread will be terminated after task done.
		// defer runtime.UnlockOSThread()
		defer wg.Done()

		f, err := os.OpenFile(nsFile, os.O_RDONLY|unix.O_CLOEXEC, 0)
		if err != nil {
			err2 = err
			return
		}
		defer f.Close()

		err = unix.Setns(int(f.Fd()), 0)

		callback()
	}()
	wg.Wait()
	return err2
}

func NsEnterContainerAndRunCodes(c *cri.Container, callback func()) error {
	if strings.HasPrefix(c.NetworkMode, "netns:") {
		netNsFile := filepath.Join(core.GetHostfs(), c.NetworkMode[len("netns:"):])
		return NsEnterAndRunCodes(netNsFile, callback)
	}

	callback()
	return nil
}

func NsEnterDial(ctx context.Context, c *cri.Container, network, addr string, dialTimeout time.Duration) (net.Conn, error) {
	var conn net.Conn
	var err error
	err2 := NsEnterContainerAndRunCodes(c, func() {
		conn, err = (&net.Dialer{Timeout: dialTimeout}).DialContext(ctx, network, addr)
	})
	if err == nil {
		err = err2
	}
	return conn, err
}
