/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/dryrun/event"
	"time"
)

type (
	// 一个日志组, 单行情况下一行日志为一个组
	LogGroup struct {
		// 第一行
		Line string
		// 多行case, 99%的case都只有一行
		Lines []string
	}
	// 日志上下文
	LogContext struct {
		// for log
		log      *LogGroup
		path     string
		pathTags map[string]string
		// 如果是基于 separator 分割的则会有该字段
		columns []string
		// map模式
		columnMap    map[string]interface{}
		logTags      map[string]interface{}
		tz           *time.Location
		event        *event.Event
		whereEvent   *event.WhereEvent
		periodStatus *PeriodStatus
		vars         map[string]interface{}

		// Value is a value related to this context.
		// It is used when doing transform.
		contextValue interface {
		}
	}
)

func (c *LogContext) GetLine() string {
	return c.log.Line
}

func (c *LogContext) GetColumnByIndex(index int) (string, error) {
	if index >= 0 && index < len(c.columns) {
		return c.columns[index], nil
	}
	return "", errors.New("GetColumnByIndex")
}

func (c *LogContext) GetColumnByName(name string) (interface{}, error) {
	if len(c.columnMap) > 0 {
		x, ok := c.columnMap[name]
		if ok {
			return x, nil
		}
	}

	if len(c.logTags) > 0 {
		return c.logTags[name], nil
	}

	return nil, nil
}

func (c *LogContext) clearData() {
	c.log = nil
	c.columns = nil
	c.columnMap = nil
	c.event = nil
	c.whereEvent = nil
	c.periodStatus = nil
	c.contextValue = nil
	c.vars = nil
	c.tz = nil
	c.logTags = nil
}

func (l *LogGroup) FirstLine() string {
	return l.Line
}

func (l *LogGroup) Add(line string) {
	if len(l.Lines) == 0 {
		l.Line = line
	}
	l.Lines = append(l.Lines, line)
}

func (l *LogGroup) SetOneLine(line string) {
	l.Line = line
	l.Lines[0] = line
}

func (l *LogGroup) reset() {
	l.Line = ""
	l.Lines = nil
}
