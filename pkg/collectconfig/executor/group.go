/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
)

type (
	XGroupBy interface {
		// TODO groupBy的结果存在哪里
		GroupNames() []string
		Execute(ctx *LogContext) ([]string, error)
		MaxKeySize() int
	}

	xGroupBy struct {
		groupNames []string
		groups     []*xGroup
		maxKeySize int
	}
	xGroup struct {
		name  string
		elect XElect
	}
)

func (x *xGroupBy) MaxKeySize() int {
	return x.maxKeySize
}

func parseGroupBy(gb *collectconfig.GroupBy) (XGroupBy, error) {
	if gb == nil {
		return nil, errors.New("collectconfig.GroupBy is nil")
	}

	// 确保名字没有重复
	groupNames := make([]string, len(gb.Groups))
	xgroups := make([]*xGroup, len(gb.Groups))
	for i, group := range gb.Groups {
		elect, err := parseElect(group.Elect)
		if err != nil {
			return nil, err
		}
		groupNames[i] = group.Name
		xgroups[i] = &xGroup{
			name:  group.Name,
			elect: elect,
		}
	}

	return &xGroupBy{
		maxKeySize: gb.MaxKeySize,
		groupNames: groupNames,
		groups:     xgroups,
	}, nil
}

func (x *xGroupBy) GroupNames() []string {
	return x.groupNames
}

func (x *xGroupBy) Execute(ctx *LogContext) ([]string, error) {
	// TODO 此处的 groupValues 显然是可以复用的, 要么挂在ctx上复用
	// 要么让用户传递进来

	groupValues := make([]string, len(x.groups))
	for i, g := range x.groups {
		s, err := g.elect.ElectString(ctx)
		if err != nil {
			return nil, err
		}
		groupValues[i] = s
	}
	return groupValues, nil
}
