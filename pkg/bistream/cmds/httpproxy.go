/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package cmds

import (
	"bytes"
	"context"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/bistream/biztypes"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	commonpb "github.com/traas-stack/holoinsight-agent/pkg/server/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	httpBodyLimit = 3 * 1024 * 1024
)

func HttpProxy(_ int32, bytes []byte) (int32, interface{}, error) {
	resp := &pb.HttpProxyResponse{
		Header: &commonpb.CommonResponseHeader{},
	}
	err := httpProxy0(bytes, resp)
	fillError(resp.Header, err)
	return biztypes.BizTypeHttpProxyResp, resp, err
}

func httpProxy0(reqBytes []byte, resp *pb.HttpProxyResponse) error {
	req := &pb.HttpProxyRequest{
		Header: &commonpb.CommonRequestHeader{},
	}
	err := proto.Unmarshal(reqBytes, req)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// TODO检查 url 是否安全
	url, err := url.ParseRequestURI(req.Url)
	if err != nil {
		return err
	}

	// 防止传入一些恶意的url
	host := url.Host
	if index := strings.IndexByte(host, ':'); index > 0 {
		host = host[:index]
	}
	if !secureHost(host) {
		return fmt.Errorf("insecure host: ip=%s ncIp=%s", host, util.GetLocalIp())
	}

	var httpReqBody io.Reader = nil

	if len(req.Body) > 0 {
		httpReqBody = bytes.NewBuffer(req.Body)
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, req.Url, httpReqBody)
	if err != nil {
		return err
	}

	begin := time.Now()
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	resp.Code = int32(httpResp.StatusCode)
	headers := make(map[string]string)
	for k := range httpResp.Header {
		headers[k] = httpResp.Header.Get(k)
	}
	resp.Headers = headers
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, httpBodyLimit))
	if err != nil {
		return err
	}
	resp.Body = body
	resp.Cost = int32(time.Now().Sub(begin).Milliseconds())

	return nil
}

// check host is secure to access
func secureHost(host string) bool {
	if host == util.GetLocalIp() {
		return true
	}
	pods := ioc.Crii.GetAllPods()
	// TODO poor performance
	for _, pod := range pods {
		if pod.IP() == host {
			return true
		}
	}
	return false
}
