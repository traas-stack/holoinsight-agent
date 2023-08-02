package netproxy

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"net/http"
	"time"
)

const (
	DefaultDialTimeout   = 2 * time.Second
	DefaultSocketTimeout = 60 * time.Second
)

type (
	Handler func(ctx context.Context, pod *cri.Pod, req *http.Request) (*http.Request, *http.Response, error)
)

var (
	handlers []Handler
)

func AddHttpProxyHandler(handler Handler) {
	handlers = append(handlers, handler)
}
