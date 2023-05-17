/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
)

type (
	PodAbsFileMatcher struct {
		Target *collecttask.CollectTarget
		Path   string
	}
)

func (m *PodAbsFileMatcher) IsDynamicMultiFiles() bool {
	return false
}

func (m *PodAbsFileMatcher) Find() ([]FatPath, int, error) {
	target := m.Target

	c, err := criutils.GetMainBizContainerE(ioc.Crii, target.GetNamespace(), target.GetPodName())
	if err != nil {
		return nil, 0, err
	}

	hostPath, err := cri.TransferToHostPath0(c, m.Path, true)
	if err != nil {
		logger.Errorz("[PodAbsFileMatcher] fail to resolve to host path container=%s path=[%s]", //
			zap.String("ns", c.Pod.Namespace),
			zap.String("pod", c.Pod.Name),
			zap.String("cid", c.Id),
			zap.String("path", m.Path))
		return nil, 0, err
	}

	return []FatPath{{Path: hostPath}}, 1, nil
}

func NewPodAbsFileMatcher(target *collecttask.CollectTarget, path string) *PodAbsFileMatcher {
	return &PodAbsFileMatcher{
		Target: target,
		Path:   path,
	}
}
