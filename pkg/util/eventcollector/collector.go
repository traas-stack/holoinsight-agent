/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package eventcollector

import (
	"context"
	"encoding/json"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

type (
	EventCollector struct {
		c        chan *Event
		exporter Exporter
		cancel   func()
	}
	Exporter func([]*Event)

	Event struct {
		Type      string
		Msg       string
		Timestamp int64
	}
)

func New(d time.Duration, e Exporter) *EventCollector {
	r := &EventCollector{
		c:        make(chan *Event, 1024),
		exporter: e,
	}

	r.collectLoop(d)

	return r
}

func (ec *EventCollector) Collect(key, msg string) {
	select {
	case ec.c <- &Event{
		Type:      key,
		Msg:       msg,
		Timestamp: util.CurrentMS(),
	}:
	default:
		logger.Warnf("[eventCollector] queue full")
	}
}

func (ec *EventCollector) collectLoop(d time.Duration) {
	util.GoWithRecover(func() {
		ctx, cancel := context.WithCancel(context.Background())
		ec.cancel = cancel

		tk := time.NewTicker(d)
		defer tk.Stop()

		var cache []*Event
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				ec.exporter(cache)
				cache = nil
			case x := <-ec.c:
				cache = append(cache, x)
			}
		}
	}, func(_ interface{}) {
		ec.collectLoop(d)
	})
}

func (ec *EventCollector) Stop() {
	logger.Infof("[eventCollector] stop")
	if ec.cancel != nil {
		ec.cancel()
	}
}

func (e Event) String() string {
	bs, _ := json.Marshal(e)
	return string(bs)
}
