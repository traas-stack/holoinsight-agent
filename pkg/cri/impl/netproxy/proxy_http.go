/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package netproxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/pkg/errors"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/cricore"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	HttpProxyAddr   = "127.0.0.1:9118"
	httpExecTimeout = 5 * time.Second
)

var (
	NETPROXY_ENABLED = os.Getenv("NETPROXY_ENABLED") == "true"
	HttpProxyURL, _  = url.Parse(fmt.Sprintf("http://%s", HttpProxyAddr))
)

func HandleHttpProxy(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if !NETPROXY_ENABLED {
		return req, nil
	}

	podIp := req.Host
	if host, _, err := net.SplitHostPort(podIp); err == nil {
		podIp = host
	}

	pod := criutils.FindFirstPodByIp(ioc.Crii, podIp)
	if pod == nil {
		return req, CreateServerInternalErrorResp(req, "unknown pod")
	}

	sandbox := pod.Sandbox
	if sandbox == nil {
		return req, CreateServerInternalErrorResp(req, "no sandbox")
	}

	if sandbox.Runtime == cri.Runc {
		return HandleHttpProxyByNsenter(pod, req, "")
	}

	pctx := context.Background()
	for _, handler := range handlers {
		preq, presp, err := handler(pctx, pod, req)
		if err != nil {
			logger.Errorz("[netproxy] handle http proxy error", zap.Error(err))
			continue
		}
		if presp != nil {
			return preq, presp
		}
	}

	return HandleHttpProxyByExec(pod, req, ctx)
}

func createNsEnterHttpClient(pod *cri.Pod) (*http.Transport, *http.Client, error) {
	return util.CreateHttpClientWithDialContext(func(ctx context.Context, network, addr string) (net.Conn, error) {
		timeout := DefaultDialTimeout
		if d, ok := ctx.Deadline(); ok {
			timeout = d.Sub(time.Now())
		}
		return cricore.NsEnterDial(ctx, pod.Sandbox, network, addr, timeout)
	})
}

func CreateServerInternalErrorResp(req *http.Request, extra string) *http.Response {
	return goproxy.NewResponse(req, "", 500, "Server Internal Error: "+extra)
}

func HandleHttpProxyByNsenter(pod *cri.Pod, req *http.Request, forceIp string) (*http.Request, *http.Response) {
	t, client, err := createNsEnterHttpClient(pod)
	if err != nil {
		return req, CreateServerInternalErrorResp(req, err.Error())
	}
	defer t.CloseIdleConnections()

	req.RequestURI = ""
	if forceIp != "" {
		if _, port, err := net.SplitHostPort(req.Host); err == nil {
			req.Host = fmt.Sprintf("%s:%s", forceIp, port)
			req.URL.Host = req.Host
		}
	}

	// TODO timeout ?
	resp, err := client.Do(req)
	if err != nil {
		return req, CreateServerInternalErrorResp(req, errors.Wrap(err, "nsenter error").Error())
	}
	if resp != nil {
		resp.Header.Add("X-Hacked-By", "holoinsight-agent ns")
	}
	return req, resp
}

func HandleHttpProxyByExec(pod *cri.Pod, req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	reqBuf := bytes.NewBuffer(nil)
	err := req.Write(reqBuf)
	if err != nil {
		logger.Errorz("[netproxy] write http request error", zap.Error(err))
		return req, CreateServerInternalErrorResp(req, "write request error")
	}

	if len(pod.Biz) == 0 {
		return req, CreateServerInternalErrorResp(req, "biz container is empty")
	}

	execCtx, cancel := context.WithTimeout(context.Background(), httpExecTimeout)
	defer cancel()
	r, err := ioc.Crii.Exec(execCtx, pod.Biz[0], cri.ExecRequest{
		Cmd:   []string{core.HelperToolPath, "httpProxy"},
		Env:   []string{"HTTPPROXY_TIMEOUT=" + httpExecTimeout.String(), "NO_JSON_OUTPUT=true"},
		Input: reqBuf,
	})
	if err != nil {
		logger.Errorz("[netproxy] cri exec error", zap.Error(err))
		return req, CreateServerInternalErrorResp(req, "cri exec error")
	}

	httpResp, err := http.ReadResponse(bufio.NewReader(r.Stdout), req)
	if err != nil {
		logger.Errorz("[netproxy] read http response error", zap.Error(err))
		return req, CreateServerInternalErrorResp(req, "")
	}

	httpResp.Header.Add("X-Hacked-By", "holoinsight-agent exec")
	return req, httpResp
}
