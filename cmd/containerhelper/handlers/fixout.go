/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package handlers

import (
	fixout2 "github.com/traas-stack/holoinsight-agent/cmd/containerhelper/handlers/fixout"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"os"
	"os/exec"
)

// fixOutHandler will run another process and encode the stdout/stderr of that process into fixOutHandler's stdout.
func fixOutHandler(action string, resp *model.Resp) error {
	// build cmd
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	stdoutr, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	errChan := make(chan error, 2)
	// encode cmd's stdout and stderr into os.Stdout
	go fixout2.CopyStream(fixout2.StdoutFd, stdoutr, errChan)
	go fixout2.CopyStream(fixout2.StderrFd, stderrr, errChan)

	wait := 2
loop:
	for {
		select {
		case <-errChan:
			wait--
			if wait == 0 {
				cmd.Wait()
				// done
				break loop
			}
		}
	}

	return nil
}
