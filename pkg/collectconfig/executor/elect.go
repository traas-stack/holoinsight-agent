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
	"regexp"
)

type (
	XElect interface {
		Init()
		Elect(ctx *LogContext) (interface{}, error)
		ElectString(ctx *LogContext) (string, error)
		ElectNumber(ctx *LogContext) (float64, error)
	}
)

func parseElect(e *collectconfig.Elect) (XElect, error) {
	xe, err := parseElect0(e)
	if err == nil {
		xe.Init()
	}
	if e.Transform != nil {
		filter, err := parseTransform(e.Transform)
		if err != nil {
			logger.Errorz("fail to parse transform", zap.Any("elect", e), zap.Error(err))
		} else {
			xe = &xElectWrap{
				inner:     xe,
				transform: filter,
			}
		}
	}
	return xe, err
}

func parseElect0(e *collectconfig.Elect) (XElect, error) {
	if e == nil {
		return nil, errors.New("elect is nil")
	}
	switch e.Type {
	case collectconfig.EElectLeftRight:
		rule := e.LeftRight
		if rule.LeftIndex < 0 {
			return nil, errors.New("leftIndex<0")
		}
		return &xLeftRight{
			rule: rule,
		}, nil
	case collectconfig.EElectRefIndex:
		return &xElectRefIndex{
			index: e.RefIndex.Index,
		}, nil
	case collectconfig.EElectRefName:
		return &xElectRefName{
			name: e.RefName.Name,
		}, nil
	case collectconfig.EElectLine:
		index := 0
		if e.Line != nil {
			index = e.Line.Index
		}
		return &xElectLine{
			index: index,
		}, nil
	case collectconfig.EElectRegexp:
		// 通过正则表达式捕获组切出一个字段
		r, err := regexp.Compile(e.Regexp.Expression)
		if err != nil {
			return nil, err
		}
		names := r.SubexpNames()
		index := e.Regexp.Index
		// 如果name非空的情况下就认为这是捕获组的名字
		if e.Regexp.Name != "" {
			index = util.StringSliceFind(names, e.Regexp.Name)
		}
		if index < 0 {
			return nil, errors.New("regexp.index < 0")
		}
		if e.Regexp.Index >= len(names) {
			return nil, errors.New("regexp.index >= len(regexp.SubexpNames())")
		}
		return &xRegexpElect{
			r:     r,
			index: index,
		}, nil
	case collectconfig.EElectRefMeta:
		value := ""
		if e.RefMeta == nil {
			return nil, errors.New("refMeta is nil")
		}
		switch e.RefMeta.Name {
		case "ip":
			value = util.GetLocalIp()
		case "hostname":
			value = util.GetHostname()
		default:
			return nil, errors.New("unsupported meta " + e.RefMeta.Name)
		}
		return &xElectRefMeta{
			value: value,
		}, nil
	case collectconfig.EElectPathVar:
		if e.PathVar == nil {
			return nil, errors.New("pathVar is nil")
		}
		return &xPathVar{name: e.PathVar.Name}, nil
	case collectconfig.EElectContext:
		return xElectContextInstance, nil
	case collectconfig.EElectRefVar:
		return &xRefVar{name: e.RefVar.Name}, nil
	}
	return nil, errors.New("unsupported elect type " + e.Type)
}
