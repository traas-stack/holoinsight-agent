package collecttaskcri

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
)

func GetTargetContainerE(i cri.Interface, target *collecttask.CollectTarget) (*cri.Container, error) {
	switch target.Type {
	case collecttask.TargetContainer:
		k8sContainerName := target.Meta["containerName"]
		return criutils.GetContainerE(i, target.GetNamespace(), target.GetPodName(), k8sContainerName)
	case collecttask.TargetPod:
		return criutils.GetMainBizContainerE(i, target.GetNamespace(), target.GetPodName())
	}
	return nil, cri.ErrNoSuchContainer
}
