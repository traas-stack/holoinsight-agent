/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package collecttask

import (
	"context"
	uuid2 "github.com/google/uuid"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// sync config interval
	syncInterval = 1 * time.Minute
	// sync config request timeout
	syncTimeout = 10 * time.Second
)

type (
	ChangeListener interface {
		// OnUpdate apply config delta to listener
		OnUpdate(*Delta)
	}

	// 管理采集配置
	// 1. 本地缓存
	// 2. 从Registry缓存
	IManager interface {
		GetAll() []*CollectTask
		Listen(listener ChangeListener)
		RemoveListen(listener ChangeListener)
	}
	Manager struct {
		// reg服务
		rs *registry.Service
		// 锁保护
		mutex sync.RWMutex
		// 我先假设配置都放内存里
		// 这里按之前设想的采用分组方式进行存储
		buckets map[string]*BucketInfo
		agentId string
		// local storage
		storage            *Storage
		listeners          []ChangeListener
		tasksCount         int
		stopSignal         *util.StopSignal
		manuallySyncOnceCh chan struct{}
	}
	BucketInfo struct {
		key   string
		state string
		// 现有的每个采集任务
		tasks map[string]*CollectTask
	}
	Delta struct {
		// 一次增量更新的uuid
		Uuid string
		Add  []*CollectTask
		Del  []*CollectTask
	}

	syncContext struct {
		Uuid     string
		Begin    time.Time
		End      time.Time
		Err      error
		StateMap map[string]string
		Changed  bool
	}
)

// 依赖 regsitry 服务
// 建议提供一个 AgentMetaService 供获取信息
func NewManager(rs *registry.Service, agentId string) (*Manager, error) {
	s, err := NewStorage("data/config.db")
	if err != nil {
		return nil, err
	}

	m := &Manager{
		rs:                 rs,
		agentId:            agentId,
		storage:            s,
		stopSignal:         util.NewStopSignal(),
		manuallySyncOnceCh: make(chan struct{}, 1),
	}
	return m, nil
}

func (m *Manager) InitLoad() {
	// 1. load from local db
	all, _ := m.storage.GetAll()
	m.buckets = all
	for _, b := range all {
		m.tasksCount += len(b.tasks)
	}

	for _, info := range all {
		for _, task := range info.tasks {
			logger.Configz("[ctm] init load", zap.Any("task", task), zap.String("content", string(task.Config.Content)))
		}
	}

	logger.Configz("[ctm] init load stat", zap.Int("buckets", len(all)), zap.Int("tasks", m.tasksCount))

	// 2. try sync from registry right now
	m.syncOnce()
}

func (m *Manager) Stop() {
	logger.Configz("[ctm] stop")
	m.stopSignal.Stop()
	m.stopSignal.WaitStopped()
	m.storage.Close()
}

// Listen to config change
func (m *Manager) Listen(listener ChangeListener) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.listeners = append(m.listeners, listener)
}

func (m *Manager) RemoveListen(listener ChangeListener) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, l := range m.listeners {
		if l == listener {
			var temp []ChangeListener
			temp = append(temp, m.listeners[:i]...)
			temp = append(temp, m.listeners[i+1:]...)
			m.listeners = temp
			m.listeners[len(m.listeners)-1] = nil
			break
		}
	}
}

func (m *Manager) GetAll() []*CollectTask {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tasks := make([]*CollectTask, 0, m.tasksCount)
	for _, b := range m.buckets {
		for _, t := range b.tasks {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

func (m *Manager) syncOnce() {
	util.WithRecover(m.syncOnce0)
}

func (m *Manager) syncOnce0() {
	syncCtx := &syncContext{
		Begin: time.Now(),
	}
	{
		if u, err := uuid2.NewUUID(); err == nil {
			syncCtx.Uuid = strings.ReplaceAll(u.String(), "-", "")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	resp, err := m.queryCollectTasks(ctx)
	if err != nil {
		logger.Configz("[ctm] GetCollectTasks error", zap.String("uuid", syncCtx.Uuid), zap.Error(err))
		return
	}

	switch resp.GetHeader().GetCode() {
	case registry.CodeOk:
		m.applyCollectTasks(syncCtx, resp)
		syncCtx.End = time.Now()
		logger.Configz("[ctm] sync once", zap.Any("ctx", syncCtx))
	default:
		syncCtx.End = time.Now()
		logger.Configz("[ctm] GetCollectTasks error", zap.String("uuid", syncCtx.Uuid), zap.Stringer("resp", resp))
	}

}

func (m *Manager) queryCollectTasks(ctx context.Context) (*pb.GetCollectTasksResponse, error) {
	bucketsHash := make(map[string]string, len(m.buckets))
	for bkey, bi := range m.buckets {
		bucketsHash[bkey] = bi.state
	}
	return m.rs.GetCollectTasks(ctx, m.agentId, bucketsHash)
}

func (m *Manager) causeAnyChange(rBuckets map[string]*BucketInfo) bool {
	// 99% 配置没有任何变化, 此时我们可以避免加锁
	oldBuckets := m.buckets

	if len(oldBuckets) != len(rBuckets) {
		return true
	}

	for bk, bi := range oldBuckets {
		if tmp, ok := rBuckets[bk]; ok {
			if bi.state != tmp.state {
				return true
			}
		} else {
			return true
		}
	}

	for bk := range rBuckets {
		if _, ok := oldBuckets[bk]; !ok {
			return true
		}
		// ok true 的case 已经在上面遍历过了
	}

	return false
}

// key: bucket key
// memoryBucket: bucket in memory
// registryBucket: bucket returned by registry
// d: collect task delta
func (m *Manager) updateBucket(memoryBucket *BucketInfo, registryBucket *BucketInfo, d *Delta) {

	for taskKey, memoryTask := range memoryBucket.tasks {
		if registryTask, ok := registryBucket.tasks[taskKey]; ok {
			if registryTask.IsDifferentWith(memoryTask) {
				d.add(registryTask)
			}
		} else {
			d.del(memoryTask)
			m.tasksCount--
		}
	}

	for taskKey, rTask := range registryBucket.tasks {
		if _, ok := memoryBucket.tasks[taskKey]; !ok {
			// 这里只需要处理!ok, ok的case上面已经处理过了
			// 这个配置需要新增
			d.add(rTask)
			m.tasksCount++
		}
	}

	// update memory bucket
	memoryBucket.state = registryBucket.state
	memoryBucket.tasks = registryBucket.tasks

	if err := m.storage.Set(memoryBucket); err != nil {
		logger.Configz("db error", zap.Error(err))
	}
}

func (m *Manager) addBucket(registryBucket *BucketInfo, d *Delta) {
	m.buckets[registryBucket.key] = registryBucket
	for _, t := range registryBucket.tasks {
		d.add(t)
	}
	m.tasksCount += len(registryBucket.tasks)
	if err := m.storage.Set(registryBucket); err != nil {
		logger.Configz("db error", zap.Error(err))
	}
}

func (m *Manager) delBucket(memoryBucket *BucketInfo, d *Delta) {
	delete(m.buckets, memoryBucket.key)
	for _, t := range memoryBucket.tasks {
		d.del(t)
	}
	m.tasksCount -= len(memoryBucket.tasks)
	if err := m.storage.Remove(memoryBucket.key); err != nil {
		logger.Configz("db error", zap.Error(err))
	}
}

// 现在还在锁内
func (m *Manager) applyCollectTasks(syncCtx *syncContext, resp *pb.GetCollectTasksResponse) {
	// rBuckets: Registry Buckets
	rBuckets, err := toBuckets(resp)
	if err != nil {
		logger.Configz("[ctm] convert registry bucket error", zap.String("uuid", syncCtx.Uuid), zap.Error(err))
		return
	}
	stateMap := make(map[string]string, len(rBuckets))
	for k, v := range rBuckets {
		stateMap[k] = v.state
	}
	syncCtx.StateMap = stateMap
	if !m.causeAnyChange(rBuckets) {
		return
	}
	syncCtx.Changed = true

	// 写锁
	m.mutex.Lock()
	defer m.mutex.Unlock()

	oldBuckets := m.buckets

	// 1. 更新manager自己内存里的配置
	// 2. 更新磁盘存储里的配置

	// 存储本次变化的增量
	d := &Delta{Uuid: syncCtx.Uuid}

	// registryBucket: Registry Bucket Info
	for _, bi := range oldBuckets {
		if rbi, ok := rBuckets[bi.key]; ok {
			if bi.state != rbi.state {
				m.updateBucket(bi, rbi, d)
				// 说明该bucket发生了变化
				// 1. 更新该bucket
				// 2. 更新在bucket到磁盘
				// 3. apply所有listener
			}
		} else {
			// 说明该bucket需要被删除
			m.delBucket(bi, d)
		}
	}

	for _, registryBucket := range rBuckets {
		if _, ok := oldBuckets[registryBucket.key]; !ok {
			// 说明该bucket需要被新增
			m.addBucket(registryBucket, d)
		}
		// ok true 的case 已经在上面遍历过了
	}

	for _, task := range d.Add {
		logger.Configz("[ctm] delta add", zap.String("uuid", syncCtx.Uuid), zap.Any("task", task), zap.String("content", string(task.Config.Content)))
	}
	for _, task := range d.Del {
		logger.Configz("[ctm] delta del", zap.String("uuid", syncCtx.Uuid), zap.Any("task", task))
	}

	// 3. 将配置(增量或者全量)apply给listener
	for _, l := range m.listeners {
		l.OnUpdate(d)
	}
}

func (m *Manager) StartListen() {
	logger.Configz("[ctm] start listen")
	go m.listenLoop()
}

func (m *Manager) MaybeSyncOnce() {
	select {
	case m.manuallySyncOnceCh <- struct{}{}:
	default:
	}
}

func getInterval(seconds int32, defaultDuration time.Duration) time.Duration {
	if seconds <= 0 {
		return defaultDuration
	}
	return time.Duration(seconds) * time.Second
}

func (m *Manager) listenLoop() {
	defer m.stopSignal.StopDone()

	resp := m.rs.GetLastControlConfigs()
	interval := syncInterval
	if resp != nil {
		interval = getInterval(resp.GetBasicConfig().GetSyncConfigsIntervalSeconds(), syncInterval)
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-m.manuallySyncOnceCh:
			m.syncOnce()
		case <-timer.C:
			m.syncOnce()
			resp := m.rs.GetLastControlConfigs()
			if resp != nil {
				interval = getInterval(resp.GetBasicConfig().GetSyncConfigsIntervalSeconds(), syncInterval)
			}
			timer.Reset(interval)
		case <-m.stopSignal.C:
			return
		}
	}
}

func (m *Manager) AddHttpFuncs() {
	http.HandleFunc("/api/agent/syncConfig", func(writer http.ResponseWriter, _ *http.Request) {
		m.MaybeSyncOnce()
		writer.Write([]byte("OK"))
	})
}

func (m *Manager) CheckTask(configKey, configVersion, targetKey, targetVersion string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, bucket := range m.buckets {
		for _, task := range bucket.tasks {
			if task.Config.Key == configKey && task.Target.Key == targetKey {
				if task.Config.Version == configVersion && task.Target.Version == targetVersion {
					return 1
				}
				return 2
			}
		}
	}
	return 0
}

func (d *Delta) add(t *CollectTask) {
	d.Add = append(d.Add, t)
}

func (d *Delta) del(t *CollectTask) {
	d.Del = append(d.Del, t)
}
