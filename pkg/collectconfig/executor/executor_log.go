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
	if c.columnMap == nil {
		return nil, nil
	}
	return c.columnMap[name], nil
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
}
