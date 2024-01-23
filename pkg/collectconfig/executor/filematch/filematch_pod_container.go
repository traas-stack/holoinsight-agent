/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/dockerutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
)

type (
	ContainerFileMatcher struct {
		Target *collecttask.CollectTarget
	}
)

func (m *ContainerFileMatcher) Find() ([]FatPath, int, error) {
	c, err := criutils.GetMainBizContainerE(ioc.Crii, m.Target.GetNamespace(), m.Target.GetPodName())
	if err != nil {
		return nil, 0, err
	}
	if c.LogPath == "" {
		return nil, 0, errors.New("empty logPath")
	}
	return []FatPath{{
		Path: c.LogPath,
		Attrs: map[string]string{
			dockerutils.AttrIsDockerJsonLog: "true",
		},
	}}, 0, nil
}

func (c *ContainerFileMatcher) IsDynamicMultiFiles() bool {
	return false
}

func NewContainerFileMatcher(target *collecttask.CollectTarget) *ContainerFileMatcher {
	return &ContainerFileMatcher{
		Target: target,
	}
}
