//go:build linux

/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cricore

import (
	"golang.org/x/sys/unix"
	"os"
	"runtime"
	"sync"
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
		// Don't call UnlockOSThread. This thread will terminated after task done.
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
