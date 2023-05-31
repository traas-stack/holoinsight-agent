/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/traas-stack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/integration/base"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/integration/alibabacloud"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/traas-stack/holoinsight-agent/pkg/util/recoverutils"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultMetricsEndpoint   = "metrics.cn-hangzhou.aliyuncs.com"
	prefix                   = "alibabacloud"
	aliyunMetricDefaultDelay = 5 * time.Minute
)

type (
	Pipeline struct {
		tenant string
		conf   *taskConf
		// TODO 记录每个指标上次成功同步到的timestamp, 然后下次从它开始继续同步
		stop                          chan struct{}
		ratelimiter                   ratelimit.Limiter
		task                          *collecttask.CollectTask
		timeCache                     *timeCache
		namespaceIntelligentFrequency map[string]*intelligentFrequency
	}
	taskConf struct {
		base.Conf
		Confs []*subConf `json:"confs"`
	}
	subConf struct {
		AccountId       string `json:"accountId"`
		AccessKeyId     string `json:"accessKeyId"`
		AccessKeySecret string `json:"accessKeySecret"`
		Range           range2 `json:"range"`
	}
	range2 struct {
		NamespaceMetrics []*namespaceMetric `json:"nameMetrics,omitempty"`
	}
	namespaceMetric struct {
		// name
		Name      string        `json:"name,omitempty"`
		Namespace string        `json:"namespace,omitempty"`
		Metrics   []*metricConf `json:"metrics,omitempty"`
	}
	metricConf struct {
		AliyunMetricName    string   `json:"aliyunMetricName,omitempty"`
		ConvertedMetricName string   `json:"convertedMetricName,omitempty"`
		Dimensions          []string `json:"dimensions,omitempty"`
		Aggregations        []string `json:"aggregations,omitempty"`
	}
)

func (p *Pipeline) Start() {
	logger.Infoz("[pipeline] [alibabacloud] task start", zap.Any("conf", util.ToJsonString(p.conf)))
	go p.taskLoop()
}

func (p *Pipeline) Stop() {
	close(p.stop)
}

func (p *Pipeline) SetupConsumer(st *api.SubTask) error {
	return nil
}

func (p *Pipeline) Update(f func(api.Pipeline)) {
	f(p)
}

func (p *Pipeline) View(f func(api.Pipeline)) {
	f(p)
}

func ParsePipeline(task *collecttask.CollectTask) (*Pipeline, error) {
	tenant := task.Target.GetTenant()
	if tenant == "" {
		return nil, errors.New("target.meta['tenant'] is nil")
	}

	conf := &taskConf{}
	err := json.Unmarshal(task.Config.Content, conf)
	if err != nil {
		return nil, err
	}
	return &Pipeline{
		task:   task,
		tenant: tenant,
		conf:   conf,
		stop:   make(chan struct{}),
		// 免费用户限速查询 https://quotas.console.aliyun.com/flow-control-products/cms/quotas
		// 默认 50次/s
		ratelimiter:                   ratelimit.New(40),
		timeCache:                     newTimeCache(),
		namespaceIntelligentFrequency: make(map[string]*intelligentFrequency),
	}, nil
}

func (p *Pipeline) flush(metrics []*model.Metric) {
	for _, metric := range metrics {
		meta.AttachSystemCommonTagsTo(metric.Tags)
	}
	gw, err := gateway.Acquire()
	if err == nil {
		func() {
			defer gateway.GatewaySingletonHolder.Release()

			size := len(metrics)
			if size > 0 {
				extensions := make(map[string]string, 1)
				extensions["agentID"] = agentmeta.GetAgentId()
				extensions["tenant"] = p.tenant
				_, err := gw.WriteMetricsV1Extension2(context.Background(), extensions, metrics)
				if err != nil {
					logger.Errorz("[pipeline] [alibabacloud] WriteMetricsV1 error", zap.String("tenant", p.tenant), zap.Int("metrics", size), zap.Error(err))
				} else {
					logger.Infoz("[pipeline] [alibabacloud] WriteMetricsV1 success", zap.String("tenant", p.tenant), zap.Int("metrics", size))
				}
			}
		}()
	} else {
		logger.Errorz("[pipeline] [alibabacloud] acquire gateway error", zap.Error(err))
	}
}

func (p *Pipeline) syncOnce() {
	var traceId string
	{
		//uuid, _ := rand.Int31()
		uuid, _ := uuid.NewRandom()
		traceId = strings.ReplaceAll(uuid.String(), "-", "")
		//traceId = cast.ToString(rand.Int31())
	}

	logger.Infoz("[pipeline] [alibabacloud] sync once start", zap.String("key", p.task.Key), zap.String("traceId", traceId))

	// 阿里云这个非常多, 要分批上报, 不然超过grpc大小

	out := make(chan *model.Metric, 1024)
	go func() {
		var metrics0 []*model.Metric
		for d := range out {
			metrics0 = append(metrics0, d)
			if len(metrics0) >= 1000 {
				p.flush(metrics0)
				metrics0 = nil
			}
		}
		if len(metrics0) > 0 {
			p.flush(metrics0)
		}
	}()

	var wg sync.WaitGroup
	var success, total, pointCount int32
	begin := time.Now()

	for _, subConf0 := range p.conf.Confs {
		subConf := subConf0

		// 先不要打印 subConf, 量太大了
		logger.Debugz("[pipeline] [alibabacloud] run",
			zap.String("key", p.task.Key), //
			zap.String("traceId", traceId),
			zap.Any("conf", subConf), //
		)

		for _, namespaceMetrics0 := range subConf.Range.NamespaceMetrics {
			namespaceMetrics := namespaceMetrics0

			var ir *intelligentFrequency
			if ir0, ok := p.namespaceIntelligentFrequency[namespaceMetrics.Namespace]; ok {
				ir = ir0
			} else {
				ir = newIntelligentFrequency(3, 10)
				p.namespaceIntelligentFrequency[namespaceMetrics.Namespace] = ir
			}

			if ir.isDown() {
				// reduce frequency
				ir.set(false)
				logger.Infoz("[pipeline] [alibabacloud] namespace no data, skip", //
					zap.String("key", p.task.Key),                       //
					zap.String("traceId", traceId),                      //
					zap.String("name", namespaceMetrics.Name),           //
					zap.String("namespace", namespaceMetrics.Namespace)) //
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				success0, total0, metrics0, err0 := p.syncOnceNamespace(traceId, subConf, namespaceMetrics)
				atomic.AddInt32(&success, success0)
				atomic.AddInt32(&total, total0)
				ir.set(len(metrics0) > 0)

				if err0 == nil {
					// 如果连续 N 次则降频
					logger.Infoz("[pipeline] [alibabacloud] sync once namespace success", //
						zap.String("key", p.task.Key),                       //
						zap.String("traceId", traceId),                      //
						zap.String("name", namespaceMetrics.Name),           //
						zap.String("namespace", namespaceMetrics.Namespace), //
						zap.Int32("success", success0),                      //
						zap.Int32("total", total0),                          //
						zap.Int("metrics", len(metrics0)))
					atomic.AddInt32(&pointCount, int32(len(metrics0)))
					for _, metric := range metrics0 {
						out <- metric
					}
				} else {
					logger.Errorz("[pipeline] [alibabacloud] sync once namespace error", //
						zap.String("key", p.task.Key),                       //
						zap.String("traceId", traceId),                      //
						zap.String("name", namespaceMetrics.Name),           //
						zap.String("namespace", namespaceMetrics.Namespace), //
						zap.Error(err0))
				}
			}()
		}
	}

	wg.Wait()
	close(out)
	cost := time.Now().Sub(begin)
	logger.Infoz("[pipeline] [alibabacloud] sync once done", //
		zap.String("key", p.task.Key),       //
		zap.String("traceId", traceId),      //
		zap.Int32("success", success),       //
		zap.Int32("total", total),           //
		zap.Int32("pointCount", pointCount), //
		zap.Int("confs", len(p.conf.Confs)), //
		zap.Duration("cost", cost))          //
}

type (
	collectOnceMetricConf struct {
		endpoint         string
		subConf          *subConf
		namespaceMetrics *namespaceMetric
		metricConf       *metricConf
		start, end       time.Time
	}
)

func (p *Pipeline) syncOnceNamespace(traceId string, subConf *subConf, namespaceMetrics *namespaceMetric) (success, total int32, metrics []*model.Metric, err error) {
	now := time.Now()
	end := now.Truncate(time.Minute).Add(-time.Minute)
	defaultStart := end.Add(-aliyunMetricDefaultDelay)

	endpoint := defaultMetricsEndpoint
	if appconfig.StdAgentConfig.Central.AlibabacloudMetricEndpoint != "" {
		endpoint = appconfig.StdAgentConfig.Central.AlibabacloudMetricEndpoint
	}
	ams := alibabacloud.NewAliyunMetricsService()

	var wg sync.WaitGroup
	var mutex sync.Mutex
	for _, metricConf0 := range namespaceMetrics.Metrics {
		metricConf := metricConf0

		// calc start
		lastTs := p.timeCache.get(namespaceMetrics.Namespace, metricConf.AliyunMetricName)
		start := lastTs
		if start.IsZero() {
			start = defaultStart
		} else {
			if end.Sub(start) > aliyunMetricDefaultDelay {
				start = defaultStart
			}
			logger.Debugz("[pipeline] [alibabacloud] use cache start time",
				zap.String("key", p.task.Key),                       //
				zap.String("traceId", traceId),                      //
				zap.String("name", namespaceMetrics.Name),           //
				zap.String("namespace", namespaceMetrics.Namespace), //
				zap.String("metric", metricConf.AliyunMetricName),   //
				zap.Time("start", start),                            //
			)
			if start == end {
				continue
			}
		}

		conf := &collectOnceMetricConf{
			endpoint:         endpoint,
			subConf:          subConf,
			namespaceMetrics: namespaceMetrics,
			metricConf:       metricConf,
			start:            start,
			end:              end,
		}
		total++
		wg.Add(1)
		// TODO 解释
		isFirstQuery := total == 1
		var firstError error
		go func() {
			defer wg.Done()
			if metrics0, err := p.collectOneMetric(traceId, ams, conf); err == nil {
				logger.Debugz("[pipeline] [alibabacloud] collect once metric success",
					zap.String("key", p.task.Key),
					zap.String("traceId", traceId),
					zap.String("name", namespaceMetrics.Name),
					zap.String("namespace", namespaceMetrics.Namespace),
					zap.String("metric", metricConf.AliyunMetricName),
					zap.Int("count", len(metrics0)),
				)

				maxTs := int64(-1)
				for _, metric := range metrics0 {
					if maxTs < metric.Timestamp {
						maxTs = metric.Timestamp
					}
				}
				if maxTs > 0 {
					// TODO We suppose that most metrics has a 1min period, ignoring very few no-1min period metrics.

					// add one period
					nextStart := time.UnixMilli(maxTs).Add(time.Minute)
					logger.Debugz("[pipeline] [alibabacloud] update cache start time",
						zap.String("key", p.task.Key),                       //
						zap.String("traceId", traceId),                      //
						zap.String("name", namespaceMetrics.Name),           //
						zap.String("namespace", namespaceMetrics.Namespace), //
						zap.String("metric", metricConf.AliyunMetricName),   //
						zap.Time("nextStart", nextStart),                    //
					)
					p.timeCache.put(namespaceMetrics.Namespace, metricConf.AliyunMetricName, nextStart)
				}

				mutex.Lock()
				metrics = append(metrics, metrics0...)
				mutex.Unlock()

				atomic.AddInt32(&success, 1)
			} else {
				if isFirstQuery {
					firstError = err
				}
				logger.Errorz("[pipeline] [alibabacloud] collect once metric error",
					zap.String("key", p.task.Key),                       //
					zap.String("traceId", traceId),                      //
					zap.String("name", namespaceMetrics.Name),           //
					zap.String("namespace", namespaceMetrics.Namespace), //
					zap.String("metric", metricConf.AliyunMetricName),   //
					zap.Error(err)) //
			}
		}()
		if total == 1 {
			wg.Wait()
			if firstError != nil {
				logger.Errorz("[pipeline] [alibabacloud] first query error stops subConf task",
					zap.String("key", p.task.Key),             //
					zap.String("traceId", traceId),            //
					zap.String("ak", subConf.AccessKeyId),     //
					zap.String("sk", subConf.AccessKeySecret), //
					zap.Any("conf", conf),                     //
					zap.Error(firstError))
				err = firstError
				return
			}
		}
	}
	wg.Wait()

	return
}

func (p *Pipeline) collectOneMetric(traceId string, ams *alibabacloud.AliyunMetricsService, conf *collectOnceMetricConf) ([]*model.Metric, error) {
	aliyunMetrics, err := ams.DescribeMetricList2(p.ratelimiter, &alibabacloud.DescribeMetricList2Request{
		Endpoint:  conf.endpoint,
		Start:     conf.start,
		End:       conf.end,
		Namespace: conf.namespaceMetrics.Namespace,
		Name:      conf.metricConf.AliyunMetricName,
		Ak:        conf.subConf.AccessKeyId,
		Sk:        conf.subConf.AccessKeySecret,
	})
	if err != nil {
		return nil, err
	}

	var metrics []*model.Metric
	// 遍历每种聚合方式 (一般只有一个, 比如 average)
	for _, aggregation := range conf.metricConf.Aggregations {
		convertedName := fmt.Sprintf("%s_%s_%s_%s", prefix, //
			strings.ReplaceAll(conf.namespaceMetrics.Name, "-", "_"), //
			conf.metricConf.ConvertedMetricName, aggregation)         //
		for _, aliyunMetric := range aliyunMetrics {
			aggregationValue, ok := aliyunMetric.Value[aggregation]
			if !ok {
				continue
			}
			convertedMetric := &model.Metric{
				Name:      convertedName,
				Tags:      aliyunMetric.Tags,
				Timestamp: aliyunMetric.Timestamp,
				Value:     aggregationValue,
			}
			metrics = append(metrics, convertedMetric)
		}
	}
	return metrics, nil
}

func (p *Pipeline) getTimer() *util.AlignTsTimer {
	internalMs := time.Minute.Milliseconds()
	switch p.conf.ExecuteRule.Type {
	case "fixedRate":
		if interval, err := util.ParseDuration(p.conf.ExecuteRule.FixedRate); err == nil && interval > 0 {
			internalMs = interval.Milliseconds()
		}
	}

	if internalMs < time.Minute.Milliseconds() {
		internalMs = time.Minute.Milliseconds()
	}
	return util.NewAlignTsTimer(internalMs, 1500, 500, 1500, false)
}

func (p *Pipeline) taskLoop() {
	timer := p.getTimer()
	defer timer.Stop()
	timer.Next()

	for {
		select {
		case <-p.stop:
			return
		case <-timer.Chan():
			recoverutils.WithRecover(p.syncOnce)
			timer.Next()
		}
	}
}
