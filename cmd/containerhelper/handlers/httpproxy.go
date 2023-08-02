package handlers

import (
	"bufio"
	"context"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultTimeout      = 3 * time.Second
	httpProxyTimeoutEnv = "HTTPPROXY_TIMEOUT"
)

func httpProxyHandler(_ string, _ *model.Resp) error {
	timeout := util.ParseDurationDefault(os.Getenv(httpProxyTimeoutEnv), defaultTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return httpProxyHandler0(ctx, os.Stdin, os.Stdout)
}

func httpProxyHandler0(ctx context.Context, in io.Reader, out io.Writer) error {
	req, err := http.ReadRequest(bufio.NewReader(in))
	if err != nil {
		return err
	}
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	req = req.WithContext(ctx)

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	httpResp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	return httpResp.Write(out)
}
