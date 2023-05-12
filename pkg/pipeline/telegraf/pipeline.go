/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package telegraf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/influxdata/telegraf"
	"github.com/spf13/cast"
	containerhelpermodel "github.com/traas-stack/holoinsight-agent/cmd/containerhelper/model"
	"github.com/traas-stack/holoinsight-agent/pkg/accumulator"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/meta"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/integration/base"
	api2 "github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	telegraf2 "github.com/traas-stack/holoinsight-agent/pkg/telegraf"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	trigger2 "github.com/traas-stack/holoinsight-agent/pkg/util/trigger"
	"go.uber.org/zap"
	"strings"
	"time"
)

type (
	Pipeline struct {
		interval   time.Duration
		trigger    trigger2.Trigger
		task       *collecttask.CollectTask
		input      interface{}
		output     *Output
		tags       map[string]string
		stop       chan struct{}
		loopStop   chan struct{}
		timerState *timerState
		transform  base.Transform
	}
	timerState struct {
		etime time.Time
		timer *time.Timer
	}
)

const (
	defaultNsEnterTimeout = 5 * time.Second
)

func (p *Pipeline) SetupConsumer(st *api.SubTask) error {
	return nil
}

func (p *Pipeline) Update(f func(api.Pipeline)) {
	f(p)
}

func (p *Pipeline) View(f func(api.Pipeline)) {
	f(p)
}

func NewPipeline(executeRule *collectconfig.ExecuteRule, task *collecttask.CollectTask, input interface{}, output *Output, tags map[string]string, transform base.Transform) (*Pipeline, error) {
	intervalMills := 0
	// 现在只支持这个
	switch executeRule.Type {
	case "fixedRate":
		intervalMills = cast.ToInt(executeRule.FixedRate)
	}
	if intervalMills <= 0 {
		intervalMills = 60_000
	}
	interval := time.Duration(intervalMills) * time.Millisecond
	trigger := trigger2.WithFixedRate(interval, 0)

	if transform.MetricPrefix == "" {
		if ip, ok := input.(api2.Input); ok {
			transform.MetricPrefix = ip.GetDefaultPrefix()
		}
	}

	return &Pipeline{
		interval:   interval,
		trigger:    trigger,
		task:       task,
		input:      input,
		output:     output,
		tags:       tags,
		stop:       make(chan struct{}),
		loopStop:   make(chan struct{}),
		timerState: &timerState{},
		transform:  transform,
	}, nil
}
func (p *Pipeline) Start() {
	go p.taskLoop()

}

func (p *Pipeline) Stop() {
	// TODO 考虑平滑
	close(p.stop)
	<-p.loopStop
}

func (p *Pipeline) taskLoop() {
	timer := p.timerState.timer
	etime := p.timerState.etime

	defer func() {
		p.timerState.etime = etime
		p.timerState.timer = timer
		close(p.loopStop)
	}()

	if timer == nil {
		etime = p.trigger.Next(nil)
		timer = time.NewTimer(etime.Sub(time.Now()))
	}

	for {
		select {
		case <-p.stop:
			return
		case <-timer.C:
			util.WithRecover(func() {
				// 移动到上个周期
				p.collectOnce(etime.Add(-p.interval))
			})
			etime = p.trigger.Next(nil)
			timer.Reset(etime.Sub(time.Now()))
		}
	}
}

func (p *Pipeline) collectOnce(metricTime time.Time) {
	begin := time.Now()
	m := &accumulator.Memory{}

	var err error
	switch i := p.input.(type) {
	case telegraf.Input:
		err = i.Gather(telegraf2.Adapt(m, "_"))
	case api2.Input:
		// In some envs, there is network isolation between holoinsight-agent pod and target pod.
		// In this case, plugins such as network detection can only be executed by entering the container through docker exec.
		// The first version of this code was written for runc, and we used the nsenter tool for this purpose.
		// But then we encountered rund, and nsenter's solution didn't work for it. At this point we can only use the standard docker.
		// Currently, some methods are still named with the 'nsenter' keyword, which will remove these couplings in the future.
		if ine, ok := p.input.(api2.InputExtNsEnter); ok && ine.NetworkMode() == api2.NetworkModePod {
			err = p.collectOnceWithNsEnter(ine, m)
		} else {
			err = i.Collect(m)
		}
	default:
		err = fmt.Errorf("unsupported input %v", p.input)
	}

	cost := time.Now().Sub(begin)

	if err != nil {
		logger.Infoz("[pipeline] collect once done", //
			zap.String("key", p.task.Config.Key),   //
			zap.String("type", p.task.Config.Type), //
			zap.Int("metrics", len(m.Metrics)),     //
			zap.Duration("cost", cost),             //
			zap.Error(err))
		return
	}

	if len(m.Metrics) == 0 {
		return
	}

	ts := metricTime.UnixMilli()

	// 不同的 targetType 附加不同的 tags

	var attachTags map[string]string

	switch p.task.Target.Type {
	case collecttask.TargetPod:
		namespace := p.task.Target.GetNamespace()
		podName := p.task.Target.GetPodName()
		if pod, ok := ioc.Crii.GetPod(namespace, podName); ok {
			attachTags = meta.ExtractPodCommonTags(pod.Pod)
		}
	case collecttask.TargetLocalhost:
		attachTags = meta.ExtractSidecarTags()
	}

	for i := range m.Metrics {
		m := m.Metrics[i]
		m.Timestamp = ts

		if m.Tags == nil {
			m.Tags = make(map[string]string)
		}
		for k, v := range p.tags {
			m.Tags[k] = v
		}
		for k, v := range attachTags {
			if v != "" {
				if _, ok := m.Tags[k]; !ok {
					m.Tags[k] = v
				}
			}
		}
	}

	if x := p.transform.MetricPrefix; x != "" {
		for _, metric := range m.Metrics {
			metric.Name = x + metric.Name
		}
	}

	if logger.IsDebugEnabled() {
		for _, m := range m.Metrics {
			logger.Debugz("[pm] digest", //
				zap.String("key", p.task.Key),               //
				zap.String("metric", m.Name),                //
				zap.Time("ts", time.UnixMilli(m.Timestamp)), //
				zap.Any("tags", m.Tags),                     //
				zap.Float64("value", m.Value),               //
			)
		}
	}

	p.output.Write(m.Metrics)

	logger.Infoz("[pipeline] collect once done", //
		zap.String("key", p.task.Config.Key),   //
		zap.String("type", p.task.Config.Type), //
		zap.Int("metrics", len(m.Metrics)),     //
		zap.Duration("cost", cost),             //
		zap.Error(err))
}

func (p *Pipeline) UpdateFrom(old api.Pipeline) {
	old2, ok := old.(*Pipeline)
	if !ok {
		return
	}
	// interval 相同的情况下才要继承
	if p.interval == old2.interval {
		p.timerState = old2.timerState
		if ext, ok := p.input.(api2.InputExt); ok {
			ext.UpdateFrom(old2.input)
		}
	}
}

func (p *Pipeline) collectOnceWithNsEnter(ine api2.InputExtNsEnter, m *accumulator.Memory) error {
	// 将请求序列化一下
	state, actionType, reqBytes, timeout, err := ine.SerializeRequest()

	if err != nil {
		return err
	}

	// 获取pod
	namespace := p.task.Target.GetNamespace()
	podName := p.task.Target.GetPodName()
	pod, ok := ioc.Crii.GetPod(namespace, podName)
	if !ok {
		return cri.NoPodError(namespace, podName)
	}

	if timeout <= 0 {
		timeout = defaultNsEnterTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout+3*time.Second)
	defer cancel()

	// 怎么证明我们确实切换到了目标 net namespace 就是用如下的方式!
	// 这段代码调试用, 我先不删除
	//if false {
	//	execResult, _ := ioc.Crii.NsEnterExec(ctx, []cri.NsEnterType{cri.NsEnter_NET}, pod.Sandbox, []string{"ifconfig"}, nil, "", bytes.NewBuffer(reqBytes))
	//	fmt.Println(execResult.Stdout.String())
	//}

	// TODO 如何知道要切换哪些ns?
	// TODO 这里硬编码了 NET
	// 这里硬编码 NET 的好处是, 我不需要切 mount, 因此我可以使用本机的bin
	// 如果我使用了 mount, 那么执行命令时可见的是容器的文件系统, 看不到我的bin, 我需要通过某种方式将bin复制进去
	// 这里使用了sandbox容器, 好处是如果主容器挂了, 那么这里依旧可以通
	// execResult, err := ioc.Crii.NsEnterExec(ctx, []cri.NsEnterType{cri.NsEnter_NET}, pod.Sandbox, []string{core.HelperToolLocalPath, actionType}, nil, "", bytes.NewBuffer(reqBytes))
	// Prefer to use docker standard API.
	execResult, err := ioc.Crii.Exec(ctx, pod.MainBiz(), cri.ExecRequest{
		Cmd:   []string{core.HelperToolPath, "inputProxy", actionType},
		Input: bytes.NewBuffer(reqBytes),
	})

	// err!=nil 说明发生系统级报错, 业务报错不会体现为 err 的
	if err != nil {
		logger.Errorz("[pipeline] nsenter collect once error", //
			zap.String("key", p.task.Key),                                       //
			zap.String("cmd", execResult.Cmd),                                   //
			zap.String("stdout", strings.TrimSpace(execResult.Stdout.String())), //
			zap.String("stderr", strings.TrimSpace(execResult.Stderr.String())), //
			zap.Error(err)) //
	} else {
		bytesResp := containerhelpermodel.RespDataBytes{}
		err = json.Unmarshal(execResult.Stdout.Bytes(), &bytesResp)
		err = ine.ProcessResponse(state, bytesResp.Data, err, m)
	}

	return err
}

func putIfBlank(m map[string]string, key, value string) {
	if v, ok := m[key]; !ok || v == "" {
		m[key] = value
	}
}
