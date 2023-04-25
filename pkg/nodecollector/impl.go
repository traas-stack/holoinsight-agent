/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package nodecollector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/cmd/containerhelper/handlers"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/k8s/nodehostname"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/model"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/output/gateway"
	"github.com/traas-stack/holoinsight-agent/pkg/server/gateway/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strings"
	"time"
)

type (
	DefaultNodeCollector struct {
		stop  chan struct{}
		input []*inputWrapper
	}
	inputWrapper struct {
		inputType string
		state     []byte
	}
)

var inputModules = []string{"cpu", "mem", "load", "traffic", "tcp", "disk", "swap", "process"}

func NewDefaultNodeCollector() *DefaultNodeCollector {
	var inputs []*inputWrapper
	for _, inputType := range inputModules {
		inputs = append(inputs, &inputWrapper{
			inputType: inputType,
			state:     nil,
		})
	}

	return &DefaultNodeCollector{
		stop:  make(chan struct{}),
		input: inputs,
	}
}

func (c *DefaultNodeCollector) Start() {
	cmd := exec.Command("cp", "-f", "bin/helper", "data/helper")
	cmd.ExtraFiles = []*os.File{}
	os.NewFile(0, "")
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(errors.Wrapf(err, "cp error: %s", string(out)))
	}
	go c.loop()
}

func (c *DefaultNodeCollector) loop() {
	interval := int64(60_000)
	timer := util.NewAlignTsTimer(interval, 1000, 0, 0, true)
	defer timer.Stop()
	nextTs := timer.Next()

	for {
		select {
		case <-c.stop:
			return
		case <-timer.Chan():
			oldNextTs := nextTs
			dataTs := nextTs/interval*interval - interval
			c.collectOnce(oldNextTs, dataTs)
			nextTs = timer.Next()
		}
	}
}

func (c *DefaultNodeCollector) collectOnce(nextTs int64, expectedTs int64) {
	logger.Infoz("[DefaultNodeCollector] collectOnce")
	begin := time.Now()

	var points []*pb.Point
	for _, iw := range c.input {
		var err error
		points, err = c.collectInputOnce(expectedTs, iw, points)
		if err != nil {
			logger.Errorz("[DefaultNodeCollector] collectInputOnce error", zap.String("inputType", iw.inputType), zap.Error(err))
		}
	}

	g, err := gateway.Acquire()
	if err != nil {
		return
	}

	defer gateway.GatewaySingletonHolder.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	g.WriteMetricsV1Extension(ctx, nil, points)

	cost := time.Now().Sub(begin)
	logger.Infoz("[DefaultNodeCollector] collectOnce", //
		zap.Time("nextTs", time.UnixMilli(nextTs)),         //
		zap.Time("expectedTs", time.UnixMilli(expectedTs)), //
		zap.Duration("cost", cost)) //
}

func (c *DefaultNodeCollector) collectInputOnce(expectedTs int64, iw *inputWrapper, points []*pb.Point) ([]*pb.Point, error) {
	inputReq := &handlers.CollectInputRequest{
		InputType: iw.inputType,
		State:     iw.state,
	}

	stdin := bytes.NewBuffer(nil)
	json.NewEncoder(stdin).Encode(inputReq)

	// TODO 改造成 nsenter 调用 ...
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//GOTRACEBACK=all
	//GODEBUG=gctrace=1,madvdontneed=1
	cmd := exec.CommandContext(ctx, "nsenter", "-t", "1", "-m", "-u", "-n", "/usr/local/holoinsight/agent/data/helper", "collectInput")
	cmd.Env = []string{"GOTRACEBACK=", "GODEBUG="}
	cmd.Stdin = stdin
	out, err := cmd.CombinedOutput()
	if err != nil {
		return points, err
	}
	helperResp := struct {
		core.HelperBaseResp
		Data interface{} `json:"data"`
	}{}
	inputResp := &handlers.CollectInputResponse{}
	helperResp.Data = inputResp
	err = json.Unmarshal(out, &helperResp)
	if err != nil {
		return points, errors.Wrapf(err, "json unmarshal error: %s", string(out))
	}
	if !helperResp.Success {
		return points, fmt.Errorf("call helper error: %s", helperResp.Message)
	}
	iw.state = inputResp.State

	prefix := "system_" + iw.inputType + "_"
	inputTypePrefix := iw.inputType + "_"
	for _, data := range inputResp.Datum {
		addCommonTags(data)
		for key, value := range data.Values {
			if strings.HasPrefix(key, inputTypePrefix) {
				key = key[len(inputTypePrefix):]
			}
			name := prefix + key
			points = append(points, &pb.Point{
				MetricName:   name,
				Timestamp:    expectedTs,
				Tags:         data.Tags,
				NumberValues: map[string]float64{"value": cast.ToFloat64(value)},
			})
		}
	}
	return points, nil
}

// TODO 重复
func addCommonTags(d *model.DetailData) {
	if d.Tags == nil {
		d.Tags = make(map[string]string)
	}
	d.Tags["ip"] = os.Getenv("NODE_IP")
	d.Tags["hostname"] = nodehostname.NodeHostname()
	d.Tags["nodename"] = os.Getenv("NODE_NAME")
}
