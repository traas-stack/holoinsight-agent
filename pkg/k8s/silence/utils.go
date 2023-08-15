/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package silence

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

func GetReadyStr(pod *v1.Pod) string {
	count := 0
	total := len(pod.Status.ContainerStatuses)
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			count++
		}
	}
	return fmt.Sprintf("%d/%d", count, total)
}

func IsReady(pod *v1.Pod) bool {
	count := 0
	total := len(pod.Status.ContainerStatuses)
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			count++
		}
	}
	return total > 0 && count == total
}

func GetController(pod *v1.Pod) (string, string) {
	for _, r := range pod.OwnerReferences {
		if r.Controller != nil && *r.Controller {
			return r.Kind, r.Name
		}
	}
	return "", ""
}
