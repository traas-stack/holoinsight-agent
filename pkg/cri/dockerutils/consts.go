/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package dockerutils

const (
	MergedDir            = "MergedDir"
	LabelDockerType      = "io.kubernetes.docker.type"
	LabelValuePodSandbox = "podsandbox"
)

func IsSandbox(labels map[string]string) bool {
	return labels[LabelDockerType] == LabelValuePodSandbox
}
