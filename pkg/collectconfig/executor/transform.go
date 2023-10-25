/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"reflect"
	"strings"
)

type (
	// A XTransformFilter is used to transform value.
	XTransformFilter interface {
		// Init initializes this filter instance, and returns error during initialization.
		Init() error
		// Filter transforms value
		Filter(ctx *LogContext) (interface{}, error)
	}
	xCompositeFilter struct {
		filters []XTransformFilter
	}
	xContextWrapFilter struct {
		inner XTransformFilter
	}
)

var typeOfTransformFilterConf = reflect.TypeOf(collectconfig.TransformFilterConf{})

// parseTransformFilter parse TransformFilterConf to XTransformFilter
func parseTransformFilter(filter *collectconfig.TransformFilterConf) (XTransformFilter, error) {
	var typeName string
	{
		rv := reflect.ValueOf(filter).Elem()
		for i := 0; i < rv.NumField(); i++ {
			if !rv.Field(i).IsNil() {
				typeName = strings.ToLower(typeOfTransformFilterConf.Field(i).Name)
				break
			}
		}
		if typeName == "" {
			return nil, errors.New("empty transform")
		}
	}

	var parsed XTransformFilter
	switch typeName {
	case "appendv1":
		parsed = &xAppendV1Filter{conf: filter.AppendV1}
	case "substringv1":
		parsed = &xSubstringV1Filter{conf: filter.SubstringV1}
	case "switchcasev1":
		parsed = &xSwitchCaseV1Filter{conf: filter.SwitchCaseV1}
	case "mappingv1":
		parsed = &xMappingV1Filter{conf: filter.MappingV1}
	case "constv1":
		parsed = &xConstFilter{value: filter.ConstV1.Value}
	case "compositev1":
		var filters = make([]XTransformFilter, 0, len(filter.CompositeV1.Filters))
		for _, sub := range filter.CompositeV1.Filters {
			subParsed, err := parseTransformFilter(sub)
			if err != nil {
				// Ignore this filter as if it is not exist. This could lead to unexpected result.
				logger.Errorz("[transform] fail to parse transform filter", zap.Any("filter", sub), zap.Error(err))
				continue
			}
			filters = append(filters, subParsed)
		}
		parsed = &xCompositeFilter{filters: filters}
	case "regexpreplacev1":
		parsed = &xRegexpReplaceFilter{conf: filter.RegexpReplaceV1}
	case "discardv1":
		parsed = &xDiscardFilter{}
	case "cleanurlv1":
		parsed = &xCleanUrlV1Filter{}
	default:
		// Ignore this filter as if it is not exist. This could lead to unexpected result.
		return nil, errors.New("unsupported transform filter " + util.ToJsonString(filter))
	}

	if err := parsed.Init(); err == nil {
		return &xContextWrapFilter{inner: parsed}, nil
	} else {
		return nil, err
	}
}

func parseTransform(conf *collectconfig.TransformConf) (XTransformFilter, error) {
	var filters []XTransformFilter
	for _, filter := range conf.Filters {
		parsed, err := parseTransformFilter(filter)
		if err != nil {
			// Ignore this filter as if it is not exist. This could lead to unexpected result.
			logger.Errorz("[transform] fail to parse transform filter", zap.Any("filter", filter), zap.Error(err))
			continue
		}
		filters = append(filters, parsed)
	}
	return &xCompositeFilter{filters: filters}, nil
}

func (x *xCompositeFilter) Init() error {
	return nil
}

func (x *xCompositeFilter) Filter(ctx *LogContext) (interface{}, error) {
	for _, filter := range x.filters {
		_, err := filter.Filter(ctx)
		if err != nil {
			// TODO verbose ?
			logger.Errorz("[transform] fail to execute filter, break the process", zap.Any("filter", filter), zap.Error(err))
			return ctx.contextValue, err
		}
	}
	return ctx.contextValue, nil
}

func (x *xContextWrapFilter) Filter(context *LogContext) (interface{}, error) {
	r, err := x.inner.Filter(context)
	if err == nil {
		context.contextValue = r
	}
	return r, err
}

func (x *xContextWrapFilter) Init() error {
	return x.inner.Init()
}
