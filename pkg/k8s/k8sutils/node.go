/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8sutils

import v1 "k8s.io/api/core/v1"

func GetNodeIP(node *v1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" {
			return address.Address
		}
	}
	return ""
}

func GetNodeHostname(node *v1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == "Hostname" {
			return address.Address
		}
	}
	return ""
}
