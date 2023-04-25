/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/storage"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/core"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/integration/alibabacloud"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/integration/base"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/sys"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/telegraf"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/input/nvidia_smi"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"runtime"
	"strings"
	"sync"
)

// pipeline manager

type (
	// 需要管理2种类型的pipelines
	Manager struct {
		mutex     sync.Mutex
		ctm       collecttask.IManager
		listener  *listenerImpl
		pipelines map[string]api.Pipeline
		s         *storage.Storage
		lsm       *logstream.Manager
	}
	listenerImpl struct {
		m *Manager
	}
)

// TODO 进一步抽象 不要直接依赖k8s
func NewManager(ctm collecttask.IManager) *Manager {
	m := &Manager{
		ctm:       ctm,
		pipelines: make(map[string]api.Pipeline),
		s:         storage.NewStorage(),
		lsm:       logstream.NewManager(),
	}
	m.listener = &listenerImpl{
		m: m,
	}
	return m
}

func (m *Manager) Start() {
	defer func() {
		m.mutex.Unlock()
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			logger.Configf("manager start error: %v\n%s", r, string(buf))
		}
	}()
	m.mutex.Lock()

	logger.Infoz("[pm] start")

	// 初始化加载所有配置

	tasks := m.getAllTasks()
	for _, task := range tasks {
		util.WithRecover(func() {
			m.processTask(task, true, true)
		})
	}

	for _, pipeline := range m.pipelines {
		pipeline.Start()
	}

	m.ctm.Listen(m.listener)
}

var telegrafStyleTaskTypes = make(map[string]bool)

func init() {
	telegrafStyleTaskTypes["jvmtask"] = true
	telegrafStyleTaskTypes["springboottask"] = true
	telegrafStyleTaskTypes["mysqltask"] = true
	telegrafStyleTaskTypes["httpcheck"] = true
	telegrafStyleTaskTypes["dialcheck"] = true
	telegrafStyleTaskTypes["obcollector"] = true
	telegrafStyleTaskTypes["gpu"] = true
}

func (m *Manager) processTask(task *collecttask.CollectTask, add bool, init bool) {
	configType := standardizeType(task.Config.Type)
	if telegrafStyleTaskTypes[configType] {
		m.processTelegrafTasks(task, add, init)
		return
	}

	switch configType {
	case "alicloudtask":
		m.processAliCloudTask(task, add, init)
	case "openmetricsscraperdto":
		// just ignore
	// SQL 风格的tasks
	case "sqltask":
		fallthrough
	case "":
		m.processSqlTask(task, add, init)
	default:
		logger.Configz("unknown config type", //
			zap.String("key", task.Key), //
			zap.String("configType", task.Config.Type))
	}
}

func standardizeType(t string) string {
	index := strings.LastIndexByte(t, '.')
	if index >= 0 {
		t = t[index+1:]
	}
	return strings.ToLower(t)
}

func (m *Manager) processSqlTask(task *collecttask.CollectTask, add, init bool) {
	sqlTask := &collectconfig.SQLTask{}
	err := json.Unmarshal(task.Config.Content, sqlTask)
	if err != nil {
		logger.Configz("[pm] parse sql task error", //
			zap.String("key", task.Key), //
			zap.Error(err))
		return
	}

	if util.StringSliceContains(commonSysTaskTypes, sqlTask.From.Type) && !strings.HasPrefix(task.Config.Key, builtinConfigPrefix) {
		logger.Infoz("[pm] ignore task with builtin task types", zap.Any("task", task))
		return
	}

	subTask := &api.SubTask{
		CT:      task,
		SqlTask: sqlTask,
	}

	if add {
		if existingPipeline, ok := m.pipelines[task.Key]; ok {
			if err := existingPipeline.SetupConsumer(subTask); err != nil {
				logger.Configz("[pm] fail to add consumer", //
					zap.String("key", task.Key), //
					zap.Error(err))              //
			}
		} else {
			// new add
			if p, err := m.createPipeline(task, sqlTask); err != nil {
				logger.Configz("[pm] create pipeline error", //
					zap.String("key", task.Key), //
					zap.Error(err))              //
			} else {
				if err := p.SetupConsumer(subTask); err != nil {
					logger.Configz("[pm] fail to add consumer", //
						zap.String("key", task.Key), //
						zap.Error(err))              //
					return
				}
				m.pipelines[task.Key] = p
				if !init {
					p.Start()
				}
			}
		}
	} else {
		p, ok := m.pipelines[task.Key]
		if !ok {
			logger.Configz("[pm] pipeline not found", zap.String("key", task.Key))
			return
		}
		p.Stop()
		delete(m.pipelines, task.Key)
	}
}

func (m *Manager) createPipeline(task *collecttask.CollectTask, sqlTask *collectconfig.SQLTask) (api.Pipeline, error) {
	if sqlTask.From == nil {
		return nil, errors.New("from is nil")
	}
	switch sqlTask.From.Type {
	case "log":
		return executor.NewPipeline(&api.SubTask{task, sqlTask}, m.s, m.lsm)
	default:
		// 仅在sidecar模式下才生效
		if appconfig.StdAgentConfig.Mode == core.AgentModeSidecar {
			return sys.NewSysPipeline(task, sqlTask)
		}
		return nil, fmt.Errorf("unsupported in mode %s", appconfig.StdAgentConfig.Mode)
	}
}

func (m *Manager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.ctm.RemoveListen(m.listener)
	logger.Infoz("[pm] stop")
}

func (m *Manager) onUpdate(delta *collecttask.Delta) {
	logger.Configz("[pm] update",
		zap.Int("add", len(delta.Add)),
		zap.Int("del", len(delta.Del)),
	)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, t := range delta.Add {
		m.processTask(t, true, false)
	}

	for _, t := range delta.Del {
		m.processTask(t, false, false)
	}
}

func (m *Manager) processAliCloudTask(task *collecttask.CollectTask, add, init bool) {
	if add {
		// 处理重复添加的case: 立即停掉 然后重建任务即可
		if p, ok := m.pipelines[task.Key]; ok {
			p.Stop()
			delete(m.pipelines, task.Key)
		}

		{
			// new add
			p, err := alibabacloud.ParsePipeline(task)
			if err != nil {
				logger.Errorz("[pm] parse AliCloud task error", //
					zap.String("key", task.Key), //
					zap.Error(err))
				return
			}
			m.pipelines[task.Key] = p
			if !init {
				p.Start()
			}
		}

	} else {
		if p, ok := m.pipelines[task.Key]; ok {
			p.Stop()
			delete(m.pipelines, task.Key)
		} else {
			logger.Errorz("[pm] no task %s", zap.String("key", task.Key))
		}
	}
}

func (m *Manager) processTelegrafTasks(task *collecttask.CollectTask, add bool, init bool) {
	if add {
		// 处理重复添加的case: 立即停掉 然后重建任务即可
		var old api.Pipeline
		var ok bool
		if old, ok = m.pipelines[task.Key]; ok {
			old.Stop()
			delete(m.pipelines, task.Key)
		}

		{
			// new add
			p, err := telegraf.ParsePipeline(task)
			if err != nil {
				logger.Errorz("[pm] parse telegraf style task error", //
					zap.String("key", task.Key), //
					zap.Error(err))
				return
			}
			if old != nil {
				p.UpdateFrom(old)
			}
			m.pipelines[task.Key] = p
			if !init {
				p.Start()
			}
		}

	} else {
		if p, ok := m.pipelines[task.Key]; ok {
			p.Stop()
			delete(m.pipelines, task.Key)
		} else {
			logger.Errorz("[pm] no task %s", zap.String("key", task.Key))
		}
	}
}

func (l *listenerImpl) OnUpdate(delta *collecttask.Delta) {
	l.m.onUpdate(delta)
}

func (m *Manager) getAllTasks() []*collecttask.CollectTask {
	tasks := m.ctm.GetAll()

	mode := appconfig.StdAgentConfig.Mode
	if mode == core.AgentModeSidecar || mode == core.AgentModeDaemonset {
		tasks = append(tasks, m.getBuiltInTasks()...)
	}
	return tasks
}

func (m *Manager) getBuiltInTasks() []*collecttask.CollectTask {
	// TODO load built-in tasks from files

	var tasks []*collecttask.CollectTask

	minuteExecuteRule := &collectconfig.ExecuteRule{
		Type:      "fixedRate",
		FixedRate: 60_000,
	}
	minuteBaseConf := &base.Conf{
		ExecuteRule: minuteExecuteRule,
	}

	mode := appconfig.StdAgentConfig.Mode
	if mode == core.AgentModeSidecar {
		for _, taskType := range commonSysTaskTypes {
			sqltask := &collectconfig.SQLTask{
				From: &collectconfig.From{
					Type: taskType,
				},
				Window: nil,
				Output: &collectconfig.Output{
					Type: "gateway",
					Gateway: &collectconfig.Gateway{
						MetricName: "system_%s",
					},
				},
				GroupBy:     &collectconfig.GroupBy{},
				ExecuteRule: minuteExecuteRule,
			}
			for _, tag := range commonSysTaskTags {
				sqltask.GroupBy.Groups = append(sqltask.GroupBy.Groups, &collectconfig.Group{
					Name: tag,
					Elect: &collectconfig.Elect{
						Type: collectconfig.EElectRefMeta,
						RefMeta: &collectconfig.ElectRegMeta{
							Name: tag,
						},
					},
				})
			}
			configKey := builtinConfigPrefix + taskType
			tasks = append(tasks, &collecttask.CollectTask{
				Key:     configKey,
				Version: "1",
				Config: &collecttask.CollectConfig{
					Key:     configKey,
					Type:    "SQLTASK",
					Version: "1",
					Content: util.ToJsonBytes(sqltask),
				},
				Target: &collecttask.CollectTarget{
					Key:     collecttask.TargetLocalhost,
					Type:    collecttask.TargetLocalhost,
					Version: "1",
					Meta:    make(map[string]string),
				},
			})
			logger.Infoz("[pipeline] [builtin] enable", zap.String("task", configKey))
		}
	}
	if mode == core.AgentModeSidecar || mode == core.AgentModeDaemonset {
		if nvidia_smi.IsNvidiaEnabled() {
			tasks = append(tasks, &collecttask.CollectTask{
				Key:     builtinConfigPrefix + "_gpu",
				Version: "1",
				Config: &collecttask.CollectConfig{
					Key:     builtinConfigPrefix + "_gpu",
					Type:    "gpu",
					Version: "1",
					Content: util.ToJsonBytes(minuteBaseConf),
				},
				Target: &collecttask.CollectTarget{
					Key:     collecttask.TargetLocalhost,
					Type:    collecttask.TargetLocalhost,
					Version: "1",
					Meta:    make(map[string]string),
				},
			})
			logger.Infoz("[pipeline] [builtin] nvidia-smi found, BUILTIN_gpu task is enabled")
		} else {
			logger.Infoz("[pipeline] [builtin] nvidia-smi not found, BUILTIN_gpu task is disabled")
		}
	}

	return tasks
}
