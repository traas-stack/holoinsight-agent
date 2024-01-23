/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"encoding/json"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/dryrun/event"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/filematch"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"time"
)

type (
	// TODO 请求体改造, 考虑这个是prod发给reg的
	DryRunRequest struct {
		Task  *collecttask.CollectTask
		Input *Input `json:"input,omitempty"`
	}
	Input struct {
		// read: 上层直接提供日志原文
		// read: 需要去pod上真实读取日志原文
		Type  string       `json:"type,omitempty"`
		Plain *Input_Plain `json:"plain,omitempty"`
		Read  *Input_Read  `json:"read,omitempty"`
	}
	Input_Plain struct {
		// 原始日志
		Lines []string `json:"lines,omitempty"`
		// Timezone to use when parsing timestamp of log lines.
		// Defaults to local timezone of agent.
		Timezone string `json:"timezone,omitempty"`
	}
	Input_Read struct {
		// 从末尾的最多读多少行
		MaxLines int `json:"maxLines,omitempty"`
	}
	DryRunResponse struct {
		Event       *event.Event   `json:"event,omitempty"`
		GroupResult []*GroupResult `json:"groupResult,omitempty"`
	}
	DryRunExecutor struct {
		RootEvent *event.Event
		request   *DryRunRequest
	}
	GroupResult struct {
		// 一个 group 表示一组相关的日志, 比如一个错误堆栈是一组日志
		// 对于非多行日志场景, 一个group只会包含一行日志
		GroupLines []string `json:"groupLines,omitempty"`
		// 是否满足(黑白名单)过滤条件, 如果为false, 那么 selectedValues 和 groupBy 都会是空的
		Paas bool `json:"paas,omitempty"`
		// 数值的切分结果
		SelectedValues map[string]interface{} `json:"selectedValues,omitempty"`
		// 维度的切分结果
		GroupBy map[string]interface{} `json:"groupBy,omitempty"`
	}
)

func NewDryRunExecutor(request *DryRunRequest) (*DryRunExecutor, error) {
	return &DryRunExecutor{
		request:   request,
		RootEvent: &event.Event{},
	}, nil
}

func (e *DryRunExecutor) Run() *DryRunResponse {
	task := e.request.Task

	// parse sql task
	sqltask := &collectconfig.SQLTask{}
	if err := json.Unmarshal(task.Config.Content, sqltask); err != nil {
		e.RootEvent.Error("parse SQLTask error %+v", err)
		return &DryRunResponse{Event: e.RootEvent}
	}

	// parse consumer
	subTask := &api.SubTask{CT: task, SqlTask: sqltask}
	consumer, err := parseConsumer(subTask)
	if err != nil {
		e.RootEvent.Error("parse consumer error %+v", err)
		return &DryRunResponse{Event: e.RootEvent}
	}
	consumer.SetStorage(storage.NewStorage())

	inputEvent := e.RootEvent.AddChild("input")
	switch e.request.Input.Type {
	case "plain":
		inputEvent.Set("type", "plain")
		e.processLines(e.RootEvent, consumer, filematch.FatPath{Path: "plain"}, e.request.Input.Plain.Lines)
	case "read":
		inputEvent.Set("type", "read")

		// TODO 还有一种方式是用户手动提供lines
		ld := NewLogDetector("debug", sqltask.From, task.Target)
		detectedPaths := ld.Detect()
		inputEvent.Info("detect %d path(s)", len(detectedPaths))

		for _, path := range detectedPaths {
			processPathEvent := e.RootEvent.AddChild("process path")
			processPathEvent.Set("path", path.Path).Set("pathTags", path.Tags)

			lines, err := logstream.ReadLastLines(path.Path, e.request.Input.Read.MaxLines)
			if err != nil {
				processPathEvent.Error("read error %+v", err)
				continue
			} else {
				processPathEvent.Info("read %d lines", len(lines))
				e.processLines(processPathEvent, consumer, path, lines)
			}
		}
	}
	return &DryRunResponse{Event: e.RootEvent}
}

func (e *DryRunExecutor) processLines(evt *event.Event, consumer *Consumer, fatpath filematch.FatPath, lines []string) {
	mockResp := &logstream.ReadResponse{
		IOStartTime: time.Now(),
		Lines:       lines,
		Path:        fatpath.Path,
	}
	mockIw := &inputWrapper{
		inputStateObj: inputStateObj{
			FatPath: fatpath,
		},
	}
	consumer.debugEvent = evt.AddChild("consume")
	consumer.consume(mockResp, mockIw)
}
