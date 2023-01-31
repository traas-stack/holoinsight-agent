package filematch

import (
	"fmt"
	"github.com/TRaaSStack/holoinsight-agent/pkg/collecttask"
	"github.com/TRaaSStack/holoinsight-agent/pkg/cri"
	"github.com/TRaaSStack/holoinsight-agent/pkg/ioc"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
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

	pod, ok := ioc.Crii.GetPod(target.GetNamespace(), target.GetPodName())
	if !ok {
		logger.Errorf("[PodAbsFileMatcher] no pod target=%s", util.ToJsonString(target))
		return nil, 0, fmt.Errorf("no pod, ns=[%s] pod=[%s]", target.GetNamespace(), target.GetPodName())
	}

	c := pod.MainBiz()
	if c == nil {
		logger.Errorz("[PodAbsFileMatcher] only supports one biz container", zap.String("path", m.Path), zap.String("pod", util.ToJsonString(pod)))
		return nil, 0, cri.ErrMultiBiz
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
