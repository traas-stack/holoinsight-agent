/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package standard

import (
	"context"
	"github.com/d5/tengo/v2"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"time"
)

type (
	scriptManager struct {
		key             string
		scripts         []string
		inited          bool
		compiledScripts []*tengo.Compiled
		targetMap       map[string]tengo.Object
	}
)

func newScriptManager(key string, scripts []string, target map[string]string) *scriptManager {
	targetMap := make(map[string]tengo.Object, len(target))
	for k, v := range target {
		targetMap[k] = &tengo.String{Value: v}
	}
	return &scriptManager{key: key, scripts: scripts, targetMap: targetMap}
}

func (m *scriptManager) ensureInited() {
	if m.inited {
		return
	}
	m.inited = true

	for _, script := range m.scripts {
		ts := tengo.NewScript([]byte(script))
		ts.Add("result", nil)
		ts.Add("tags", nil)
		ts.Add("target", nil)
		ts.Add("divide", DivideFunc)
		if compiled, err := ts.Compile(); err == nil {
			m.compiledScripts = append(m.compiledScripts, compiled)
		} else {
			logger.Errorz("[pipeline] fail to compile script", zap.String("key", m.key), zap.String("script", script), zap.Error(err))
		}
	}
}

func (m *scriptManager) run(metrics []*model.Metric) []*model.Metric {
	m.ensureInited()
	if len(m.compiledScripts) == 0 {
		return metrics
	}
	var groupByTags = make(map[string][]*model.Metric)
	for _, metric := range metrics {
		key := util.BuildTagsKey(metric.Tags)
		groupByTags[key] = append(groupByTags[key], metric)
	}

	var modifiedMetrics []*model.Metric

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for _, metrics := range groupByTags {

		resultMap := map[string]tengo.Object{}
		for _, metric := range metrics {
			resultMap[metric.Name] = &tengo.Float{Value: metric.Value}
		}

		tagsMap := toTengoMapStringString(metrics[0].Tags)

		// execute scripts
		for _, script := range m.compiledScripts {
			script.Set("result", resultMap)
			script.Set("tags", tagsMap)
			script.Set("target", m.targetMap)
			script.RunContext(ctx)
			script.Set("result", nil)
			script.Set("tags", nil)
			script.Set("target", nil)
		}

		tags := toGoMapStringString(tagsMap)

		for name, v := range resultMap {
			f64, ok := tengo.ToFloat64(v)
			if !ok {
				continue
			}
			modifiedMetrics = append(modifiedMetrics, &model.Metric{
				Name:      name,
				Tags:      tags,
				Timestamp: metrics[0].Timestamp,
				Value:     f64,
			})
		}
	}
	return modifiedMetrics
}

func DivideFunc(args ...tengo.Object) (tengo.Object, error) {
	m := args[0]

	var leftv float64
	var rightv float64

	{
		left, err := m.IndexGet(args[1])
		if err == nil {
			leftv, _ = tengo.ToFloat64(left)
		}
	}

	{
		right, err := m.IndexGet(args[2])
		if err == nil {
			rightv, _ = tengo.ToFloat64(right)
		}
	}

	if rightv == 0 {
		return tengo.FromInterface(0.0)
	}

	return tengo.FromInterface(leftv / rightv)
}

func toTengoMapStringString(tags map[string]string) map[string]tengo.Object {
	tagsMap := map[string]tengo.Object{}
	for k, v := range tags {
		tagsMap[k] = &tengo.String{Value: v}
	}
	return tagsMap
}

func toGoMapStringString(tagsMap map[string]tengo.Object) map[string]string {
	tags := make(map[string]string, len(tagsMap))
	for k, v := range tagsMap {
		if s, ok := tengo.ToString(v); ok {
			tags[k] = s
		}
	}
	return tags
}
