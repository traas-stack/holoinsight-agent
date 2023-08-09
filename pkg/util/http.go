/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"context"
	"net"
	"net/http"
)

// CreateHttpClientWithDialContext create a httpclient with dialcontext func
func CreateHttpClientWithDialContext(dial func(ctx context.Context, network, addr string) (net.Conn, error)) (*http.Transport, *http.Client, error) {
	t := &http.Transport{
		DialContext:         dial,
		DisableKeepAlives:   true,
		MaxIdleConnsPerHost: -1,
	}
	client := &http.Client{
		Transport: t,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	return t, client, nil
}
