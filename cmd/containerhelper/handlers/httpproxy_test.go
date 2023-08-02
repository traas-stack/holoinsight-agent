package handlers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestHttpProxy(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://www.httpbin.org/get?a=1", nil)
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBuffer(nil)
	err = req.Write(buf)
	if err != nil {
		panic(err)
	}

	fmt.Println("HTTP request:")
	fmt.Println(buf.String())

	outBuf := bytes.NewBuffer(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = httpProxyHandler0(ctx, buf, outBuf)
	if err != nil {
		panic(err)
	}

	httpResp, err := http.ReadResponse(bufio.NewReader(outBuf), req)
	if err != nil {
		panic(err)
	}
	fmt.Println(httpResp.StatusCode)
	bs, err := io.ReadAll(httpResp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println("HTTP response:")
	fmt.Println(string(bs))
}
