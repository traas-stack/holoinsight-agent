/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package dockerutils

import (
	"context"
	"github.com/docker/docker/api/types"
	dockersdk "github.com/docker/docker/client"
	"strings"
	"time"
)

const (
	pingTimeout = 3 * time.Second
)

func NewDockerClient(host string) (*dockersdk.Client, types.Ping, error) {
	docker, err := dockersdk.NewClientWithOpts(dockersdk.WithHost(host))
	if err != nil {
		return nil, types.Ping{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	pingResp, err := docker.Ping(ctx)
	if err == nil {
		// HACK: for pouch, set APIVersion to 1.25
		// which is required by github.com/docker/docker/client/container_exec.go#ContainerExecCreate
		if strings.HasSuffix(host, "pouchd.sock") {
			pingResp.APIVersion = "1.25"
		}
		docker.NegotiateAPIVersionPing(pingResp)
	}
	return docker, pingResp, err
}
