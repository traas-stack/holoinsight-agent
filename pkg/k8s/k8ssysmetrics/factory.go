/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8ssysmetrics

import (
	"os"
)

func GetNewPodSystemResourceCollector() string {
	if s := os.Getenv("HI_K8S_SYS_COLLECTOR"); s != "" {
		return s
	}
	return "cadvisor"
}
