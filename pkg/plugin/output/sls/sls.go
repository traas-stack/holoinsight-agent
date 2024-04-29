/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package sls

import (
	"fmt"
	aliyunsls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"net/http"
	"time"
)

var (
	dnsCacheHelper *util.DnsCacheHelper
	slsHttpclient  *http.Client
	slsOutput      = &SLSOutput{clientCache: make(map[string]*aliyunsls.Client)}
)

func init() {
	dnsCacheHelper = util.NewDnsCacheHelper()
	dnsCacheHelper.Start()
	slsHttpclient = dnsCacheHelper.NewHttpClient()
}

type (
	SLSOutput struct {
		clientCache map[string]*aliyunsls.Client
	}
)

func GetOutPut() *SLSOutput {
	return slsOutput
}

func (c *SLSOutput) WriteMetricsV1(metrics []*model.Metric, extension output.Extension) {
}

func NewSLSOutput() (output.Output, error) {
	return slsOutput, nil
}

func (c *SLSOutput) WriteBatchAsync(configKey, targetKey, metricName string, array []*model.DetailData) error {
	return nil
}

func (c *SLSOutput) WriteBatchV4(configKey, targetKey, metricName string, array []*model.DetailData, _ *output.PeriodCompleteness) error {
	return nil
}

func (c *SLSOutput) Start() {
}

func (c *SLSOutput) Stop() {
}

func (c *SLSOutput) WriteToSLS(configKey, targetKey string, table *model.Table, slsConfig *collectconfig.SlsConfig) error {
	if slsConfig == nil {
		return nil
	}
	client := c.getSLSClient(slsConfig)

	// Create a log group
	logGroup := &aliyunsls.LogGroup{}

	//Create log contents
	for _, row := range table.Rows {
		contents := make([]*aliyunsls.LogContent, 0)
		for i, value := range row.TagValues {
			content := &aliyunsls.LogContent{
				Key:   proto.String(table.Header.TagKeys[i]),
				Value: proto.String(value),
			}
			contents = append(contents, content)
		}
		// Create a log entry
		log := &aliyunsls.Log{
			Time:     proto.Uint32(uint32(time.Now().Unix())),
			Contents: contents,
		}

		// Add log to log group
		logGroup.Logs = append(logGroup.Logs, log)
		// PUT log to SLS
		err := client.PutLogs(slsConfig.Project, slsConfig.Logstore, logGroup)
		if err != nil {
			logger.Errorz("detail log write to sls error", zap.String("configKey", configKey), zap.String("targetKey", targetKey), zap.Error(err))
			return err
		}
	}

	logger.Infoz("detail log write to sls success", zap.String("configKey", configKey), zap.String("targetKey", targetKey), zap.Int("count", len(table.Rows)))
	return nil
}

func (c *SLSOutput) buildKey(endpoint, project, logstore string) string {
	return fmt.Sprintf("%s/%s/%s", endpoint, project, logstore)
}

func (c *SLSOutput) getSLSClient(slsConfig *collectconfig.SlsConfig) *aliyunsls.Client {
	key := c.buildKey(slsConfig.Endpoint, slsConfig.Project, slsConfig.Logstore)
	var client *aliyunsls.Client
	if v, ok := c.clientCache[key]; ok {
		client = v
	} else {
		client = &aliyunsls.Client{
			Endpoint:        slsConfig.Endpoint,
			AccessKeyID:     slsConfig.AK,
			AccessKeySecret: slsConfig.SK,
			RequestTimeOut:  5 * time.Second,
			RetryTimeOut:    5 * time.Second,
			HTTPClient:      slsHttpclient,
		}
		c.clientCache[key] = client
	}
	return client
}
