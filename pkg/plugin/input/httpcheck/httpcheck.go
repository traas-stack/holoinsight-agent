/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package httpcheck

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"io"
	"net/http"
	"regexp"
	"time"
)

type (
	// 我们的插件必须是最少依赖的, 可以被直接运行的
	Config struct {
		URL            string            `json:"url"`
		Headers        map[string]string `json:"headers"`
		Timeout        time.Duration     `json:"timeout"`
		Method         string            `json:"method"`
		SuccessCodes   []int             `json:"successCodes"`
		SuccessRegexps []*regexp.Regexp  `json:"successRegexps"`
		BodyLimit      int64             `json:"bodyLimit"`
		NetworkMode    string            `json:"networkMode"`
	}
	Input struct {
		Config *Config
		inited bool
	}
)

const (
	helperActionType = "httpcheck"
	defaultTimout    = 3 * time.Second
	MaxBodyLimit     = 10 * 1024 * 1024
)

func (i *Input) GetDefaultPrefix() string {
	return "httpcheck_"
}

func (i *Input) NetworkMode() string {
	return i.Config.NetworkMode
}

func (i *Input) SerializeRequest() (interface{}, string, []byte, time.Duration, error) {
	configBytes, err := json.Marshal(i.Config)
	if err != nil {
		return nil, "", nil, 0, err
	}
	return nil, helperActionType, configBytes, i.getTimeout(), nil
}

func (i *Input) ExecuteRequest(configBytes []byte) ([]byte, error) {
	config := &Config{}
	err := json.Unmarshal(configBytes, config)
	if err != nil {
		return nil, err
	}
	i.Config = config

	ma := api.NewMemoryAccumulator()
	err = i.Collect(ma)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ma.Metrics)
}

func (i *Input) ProcessResponse(_ interface{}, respBytes []byte, err error, accumulator api.Accumulator) error {
	// TODO 异常处理分2种
	// 1. nsenter 调用上的 OS 级错误: 我们也归类到失败上, 这也才能引起注意
	// 2. 业务错误: 确实该我们处理

	if err != nil {
		return err
	}

	return api.NsEnterHelpProcesResponse(respBytes, accumulator)
}

func (i *Input) Collect(accumulator api.Accumulator) error {
	timeout := i.getTimeout()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	begin := time.Now()
	resp, err := i.executeHttpRequest(ctx)

	if err != nil {
		defer resp.Body.Close()
	}

	// 实践中我发现 cost == 0ms 为什么? 因为速度太快了 < 1ms

	return i.checkHttpResponse(resp, begin, err, accumulator)
}

func (i *Input) executeHttpRequest(ctx context.Context) (*http.Response, error) {
	url := i.Config.URL

	if url == "" {
		return nil, errors.New("url is empty")
	}

	method := i.Config.Method
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range i.Config.Headers {
		req.Header.Set(k, v)
	}

	return (&http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}).Do(req)
}

func (i *Input) getTimeout() time.Duration {
	timeout := defaultTimout
	if i.Config.Timeout > 0 {
		timeout = i.Config.Timeout
	}
	return timeout
}

func (i *Input) checkHttpResponse(resp *http.Response, begin time.Time, err error, accumulator api.Accumulator) error {
	// 能走到这里说明 http 已经正常返回了, 但code不一定是200

	// 默认是true
	up := true

	if err == nil {
		code := resp.StatusCode
		if len(i.Config.SuccessCodes) > 0 {
			contains := false
			for _, successCode := range i.Config.SuccessCodes {
				if code == successCode {
					contains = true
					break
				}
			}
			up = contains
		}

		if up && len(i.Config.SuccessRegexps) > 0 {

			limit := i.Config.BodyLimit
			if limit > MaxBodyLimit {
				limit = MaxBodyLimit
			}

			bs, err := io.ReadAll(io.LimitReader(resp.Body, limit))
			if err != nil {
				return err
			}

			content := string(bs)
			contains := false
			for _, successRegexp := range i.Config.SuccessRegexps {
				if len(successRegexp.FindStringSubmatchIndex(content)) > 0 {
					contains = true
					break
				}
			}

			up = contains
		}
	} else {
		up = false
	}

	// 当请求头都返回时 http.Do 就返回了
	// 然后 body 是需要lazy地读网络流的, 如果你只读了一点或者没读, 那么相当于统计的cost会少一些
	// TODO 我们需要把 body 读完吗 resp.Body.Close() 里具体做了啥
	// io.Copy(io.Discard, resp.Body)
	// 但也要避免恶意攻击: 如果以后我们开放到公网给用户使用了, 并且提供了指定URL curl的功能, 假设是跑在我们中心化集群上
	// 用户可以在这里放一个非常大的文件的url, 导致我们去下载
	// TODO 设置了context timeout, 但下载是顺畅的但是速度慢, 达到超时了, 下载会中断吗? 从意义上来看是会的

	tags := map[string]string{}
	accumulator.AddMetric(&model.Metric{
		Name:      "up",
		Tags:      tags,
		Timestamp: 0,
		Value:     api.BoolToFloat64(up),
	})
	accumulator.AddMetric(&model.Metric{
		Name:      "down",
		Tags:      tags,
		Timestamp: 0,
		Value:     api.BoolToFloat64(!up),
	})

	cost := time.Now().Sub(begin)
	accumulator.AddMetric(&model.Metric{
		Name:      "cost",
		Tags:      tags,
		Timestamp: 0,
		Value:     float64(cost.Milliseconds()),
	})

	return nil
}

func (i *Input) GenerateErrorMetrics(a api.Accumulator) {
	a.AddMetric(&model.Metric{
		Name:  "up",
		Tags:  make(map[string]string),
		Value: 0,
	})
	a.AddMetric(&model.Metric{
		Name:  "down",
		Tags:  make(map[string]string),
		Value: 1,
	})
}
