/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package nvidia_smi

import (
	"encoding/xml"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"os/exec"
	"sync"
)

type (
	NvidiaSmiLog struct {
		GPU []NvidiaSmiLog_Gpu `xml:"gpu"`
	}
	NvidiaSmiLog_Gpu struct {
		ProductName string `xml:"product_name"`
		Uuid        string `xml:"uuid"`
		MinorNumber string `xml:"minor_number"`
	}
)

var (
	nvidiaSmiLog         = &NvidiaSmiLog{}
	initNvidiaSmiLogOnce = sync.Once{}
)

func GetNvidiaSmiLog() *NvidiaSmiLog {
	initNvidiaSmiLogOnce.Do(initNvidiaSmiLog)
	return nvidiaSmiLog
}

func initNvidiaSmiLog() {
	if !IsNvidiaEnabled() {
		return
	}
	var cmd *exec.Cmd
	switch appconfig.StdAgentConfig.Mode {
	case core.AgentModeDaemonset:
		cmd = exec.Command("chroot", core.GetHostfs(), "nvidia-smi", "-q", "-x")
	case core.AgentModeSidecar:
		cmd = exec.Command("nvidia-smi", "-q", "-x")
	default:
		return
	}

	b, err := cmd.CombinedOutput()
	if err != nil {
		// return err
		return
	}
	err = xml.Unmarshal(b, nvidiaSmiLog)
	if err != nil {
		return
	}
}
