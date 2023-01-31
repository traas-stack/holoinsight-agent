package springboot

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	telegraf2 "github.com/TRaaSStack/holoinsight-agent/pkg/telegraf"
	"context"
	"encoding/json"
	"fmt"
	"github.com/influxdata/telegraf"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	metricPrefix   = "springboot"
	maxConcurrency = 16
	defaultTimeout = 3 * time.Second
)

type (
	SpringBoot struct {
		// 携带上 方便排查问题
		TaskKey    string
		BaseUrl    string
		HttpClient *http.Client
		// 并发度
		Concurrency int
		state       *springBootState
	}
	springBootState struct {
		cache1 *sync.Map
		cache2 *sync.Map
	}
	actuatorMetricsResp struct {
		Names []string `json:"names"`
	}
	actuatorMetricDetail struct {
		Name         string `json:"name"`
		Measurements []*struct {
			Statistic string  `json:"statistic"`
			Value     float64 `json:"value"`
		} `json:"measurements"`
	}
)

func (s *SpringBoot) DebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"baseUrl": s.BaseUrl,
	}
}

func (s *SpringBoot) SampleConfig() string {
	return ""
}

func (s *SpringBoot) metricUrl() string {
	if strings.HasSuffix(s.BaseUrl, "/") {
		return s.BaseUrl + "metrics"
	} else {
		return s.BaseUrl + "/metrics"
	}
}

func noRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func (s *SpringBoot) httpClient() *http.Client {
	return &http.Client{
		CheckRedirect: noRedirect,
	}
}

func (s *SpringBoot) concurrency() int {
	concurrency := s.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > maxConcurrency {
		concurrency = maxConcurrency
	}
	return concurrency
}

func (s *SpringBoot) Gather(accumulator telegraf.Accumulator) error {
	if s.state == nil {
		s.state = &springBootState{
			cache1: &sync.Map{},
			cache2: &sync.Map{},
		}
	}
	defer func() {
		s.state.cache1 = s.state.cache2
		s.state.cache2 = &sync.Map{}
	}()

	url := s.metricUrl()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		content, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bad response code=[%d] content=[%s]", resp.StatusCode, string(content))
	}

	actuatorMetricsResp := &actuatorMetricsResp{}
	err = json.NewDecoder(resp.Body).Decode(actuatorMetricsResp)
	if err != nil {
		return err
	}

	concurrency := s.concurrency()
	semaphore := make(chan struct{}, concurrency)
	var out = make(chan *telegraf2.Memory, len(actuatorMetricsResp.Names))
	var wg sync.WaitGroup

	for _, name0 := range actuatorMetricsResp.Names {
		name := name0
		metricUrl := url + "/" + name
		semaphore <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
				<-semaphore
			}()

			m := &telegraf2.Memory{}
			if err := s.getMetricsFromUrl(metricUrl, name, m); err != nil {
				logger.Errorz("[springboot] get metrics error", //
					zap.String("url", metricUrl),
					zap.Error(err))
			} else {
				if logger.DebugEnabled {
					logger.Debugz("[springboot] debug", //
						zap.String("metricUrl", metricUrl), //
						zap.Int("metrics", len(m.Metrics)), //
					)
				}
			}
			out <- m
		}()
	}
	wg.Wait()
	close(out)
	for memory := range out {
		for _, metric := range memory.Metrics {
			accumulator.AddMetric(metric)
		}
	}
	return nil
}

func (s *SpringBoot) getMetricsFromUrl(metricUrl, springbootMetricName string, accumulator telegraf.Accumulator) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// TODO 不要跟入 redirect
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metricUrl, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	content := string(b)
	if logger.DebugEnabled {
		logger.Debugz("[springboot] http", //
			zap.String("metricUrl", metricUrl), //
			zap.Int("code", resp.StatusCode),   //
			zap.String("content", content))
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status error, code=%d content=%s", resp.StatusCode, content)
	}

	actuatorMetricDetailModel := &actuatorMetricDetail{}
	err = json.Unmarshal(b, actuatorMetricDetailModel)
	if err != nil {
		return err
	}

	if len(actuatorMetricDetailModel.Measurements) == 0 {
		return nil
	}
	s.state.cache2.Store(springbootMetricName, actuatorMetricDetailModel)

	var last *actuatorMetricDetail
	if loaded, ok := s.state.cache1.Load(springbootMetricName); ok {
		last = loaded.(*actuatorMetricDetail)
	} else {
		logger.Debugz("[springboot] cache1 nil", zap.String("metricUrl", metricUrl), zap.String("key", s.TaskKey))
		return nil
	}

	fields := make(map[string]interface{}, len(actuatorMetricDetailModel.Measurements))

	telegrafMeasurementName := metricPrefix + "." + actuatorMetricDetailModel.Name
	for _, measurement := range actuatorMetricDetailModel.Measurements {
		statistic := strings.ToLower(measurement.Statistic)

		switch measurement.Statistic {
		case "COUNT":
		case "TOTAL_TIME":
		default:
			fields[statistic] = measurement.Value
			continue
		}

		for _, lastM := range last.Measurements {
			if measurement.Statistic == lastM.Statistic {
				switch measurement.Statistic {
				case "COUNT":
					fields[statistic] = measurement.Value - lastM.Value
				case "TOTAL_TIME":
					fields[statistic] = measurement.Value - lastM.Value
				default:
					continue
				}
				break
			}
		}
	}

	tags := make(map[string]string)

	accumulator.AddFields(telegrafMeasurementName, fields, tags, time.Now())

	return nil
}

func (s *SpringBoot) UpdateFrom(i interface{}) {
	if s == i {
		return
	}
	s2, ok := i.(*SpringBoot)
	if !ok {
		return
	}
	s.state = s2.state
}
