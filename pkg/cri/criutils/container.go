/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package criutils

import "github.com/traas-stack/holoinsight-agent/pkg/cri"

// GetMainBizContainerE get main biz container for pod
func GetMainBizContainerE(i cri.Interface, ns string, pod string) (*cri.Container, error) {
	p, err := i.GetPod(ns, pod)
	if err != nil {
		return nil, err
	}
	return p.MainBizE()
}

func GetContainerE(i cri.Interface, ns, pod, k8sContainerName string) (*cri.Container, error) {
	p, err := i.GetPod(ns, pod)
	if err != nil {
		return nil, err
	}

	return p.GetContainer(k8sContainerName)
}
