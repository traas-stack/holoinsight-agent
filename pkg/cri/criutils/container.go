package criutils

import "github.com/traas-stack/holoinsight-agent/pkg/cri"

// GetMainBizContainerE get main biz container for pod
func GetMainBizContainerE(i cri.Interface, ns string, pod string) (*cri.Container, error) {
	p, err := i.GetPodE(ns, pod)
	if err != nil {
		return nil, err
	}
	return p.MainBizE()
}
