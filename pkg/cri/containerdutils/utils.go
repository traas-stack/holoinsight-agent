package containerdutils

import (
	"context"
	"errors"
	"github.com/containerd/containerd"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"time"
)

// NewClientFromEnv create a containerd client based on the agreed environment configuration information.
func NewClientFromEnv(addrs ...string) (*containerd.Client, containerd.Version, error) {
	// Default to two well known containerd sock
	defaultAddrs := append(addrs, "/run/containerd/containerd.sock", "/run/k3s/containerd/containerd.sock")

	addr := core.FindFirstSockInHostfs("CONTAINERD_SOCK", defaultAddrs...)
	if addr == "" {
		return nil, containerd.Version{}, errors.New("no containerd sock")
	}

	client, err := containerd.New(addr)
	if err != nil {
		return nil, containerd.Version{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	versionResp, err := client.Version(ctx)
	if err != nil {
		return nil, containerd.Version{}, err
	}
	return client, versionResp, nil
}
