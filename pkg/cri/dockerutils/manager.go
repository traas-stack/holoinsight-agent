package dockerutils

import (
	"context"
	"github.com/docker/docker/api/types"
	dockersdk "github.com/docker/docker/client"
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

	// 协商 docker api 版本
	pingResp, err := docker.Ping(ctx)
	if err == nil {
		docker.NegotiateAPIVersionPing(pingResp)
	}
	return docker, pingResp, err
}
