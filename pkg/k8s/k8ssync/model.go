/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package k8ssync

import (
	regpb "github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
)

type (
	Resource struct {
		Namespace   string            `json:"namespace"`
		Name        string            `json:"name"`
		App         string            `json:"app"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
		Ip          string            `json:"ip"`
		Hostname    string            `json:"hostname"`
		HostIP      string            `json:"hostIP"`
		Status      string            `json:"status"`
		Spec        map[string]string `json:"spec"`
	}
	FullSyncRequest struct {
		Apikey    string      `json:"apikey"`
		Type      string      `json:"type"`
		Workspace string      `json:"workspace"`
		Cluster   string      `json:"cluster"`
		Resources []*Resource `json:"resources"`
	}
	DeltaSyncRequest struct {
		Apikey    string      `json:"apikey"`
		Type      string      `json:"type"`
		Workspace string      `json:"workspace"`
		Cluster   string      `json:"cluster"`
		Add       []*Resource `json:"add"`
		Del       []*Resource `json:"del"`
	}
)

func convertToPbResource(resource *Resource) *regpb.MetaSync_Resource {
	return &regpb.MetaSync_Resource{
		Name:        resource.Name,
		Namespace:   resource.Namespace,
		Labels:      resource.Labels,
		Annotations: resource.Annotations,
		App:         resource.App,
		Ip:          resource.Ip,
		Hostname:    resource.Hostname,
		HostIP:      resource.HostIP,
		Spec:        resource.Spec,
	}
}

func convertToPbResourceSlice(resources []*Resource) []*regpb.MetaSync_Resource {
	ret := make([]*regpb.MetaSync_Resource, len(resources))
	for i, resource := range resources {
		ret[i] = convertToPbResource(resource)
	}
	return ret
}
