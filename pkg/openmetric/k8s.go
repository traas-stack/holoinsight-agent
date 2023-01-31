package openmetric

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	config2 "github.com/prometheus/common/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"net/http"
	"os"
)

type (
	MyK8sConfig     struct{}
	MyK8sDiscoverer struct {
		inner discovery.Discoverer
	}
)

func init() {
	http.Handle("/metrics", promhttp.Handler())
}

func (m *MyK8sDiscoverer) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	hihack := make(chan []*targetgroup.Group, cap(up))
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case g := <-hihack:
				// TODO 过滤, 只保留本机
				// 转发给上层
				up <- g
				continue
			}
		}
	}()

	m.inner.Run(ctx, hihack)
}

func (m *MyK8sConfig) Name() string {
	return "myk8s"
}

func (m *MyK8sConfig) NewDiscoverer(options discovery.DiscovererOptions) (discovery.Discoverer, error) {
	sdc := &kubernetes.SDConfig{
		Role: "pod",
		HTTPClientConfig: config2.HTTPClientConfig{
			TLSConfig: config2.TLSConfig{
				CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			},
			BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		},
		Selectors: []kubernetes.SelectorConfig{
			{
				Role:  "pod",
				Label: "",
				Field: fmt.Sprintf("spec.nodeName==%s", os.Getenv("NODE_NAME")),
			},
		},
	}
	d, err := sdc.NewDiscoverer(options)
	if err != nil {
		return nil, err
	}
	return &MyK8sDiscoverer{inner: d}, nil
}
