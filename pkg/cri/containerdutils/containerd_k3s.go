/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package containerdutils

// K3s is a commonly used distribution of K8s and we have built-in support for it.
const (
	K3sDefaultStateDir = "/run/k3s/containerd"
	K3sDefaultAddress  = "/run/k3s/containerd/containerd.sock"
)
