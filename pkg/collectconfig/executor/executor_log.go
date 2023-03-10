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
		columnMap  map[string]interface{}
		tz         *time.Location
		event      *event.Event
		whereEvent *event.WhereEvent
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
}
