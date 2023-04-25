/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package pouch

const (
	LabelPouchType    = "io.kubernetes.pouch.type"
	LabelValueSandbox = "sandbox"
)

func IsSandbox(labels map[string]string) bool {
	return labels[LabelPouchType] == LabelValueSandbox
}
