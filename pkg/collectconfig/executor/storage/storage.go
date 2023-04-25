/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package storage

import (
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"sync"
	"time"
)

type (
	// 存储
	Storage struct {
		// 包含多条时间线
		// TODO 用 sync.Map 还是自己用读写锁
		timelines map[string]*Timeline
		mutex     sync.RWMutex
	}
	Timeline struct {
		// 时间线的key或者name
		Key      string
		mutex    sync.RWMutex
		interval int64
		capacity int64
		// 环形数组按时间轮转
		// 时间线下的
		shards   []*Shard
		dict     map[string]string
		refCount int
	}
	// 一条时间线在某一个时间点的所有值的集合
	Shard struct {
		ts     int64
		no     int64
		points map[string]*Point
		// 如果为true说明该shard已经冻结, 比如已经emit过, 然后又收到旧数据,可以用于发现延迟日志
		Frozen bool
		// 有一些数据是shard粒度的, 并不需要做到 points 粒度
		Data interface{}
		// Data2 field is for extension
		Data2 interface{}
	}
	Point struct {
		// 时间戳
		Timestamp int64
		KeyNames  []string
		// 维度值, 维度名这里是不关心的
		Keys       []string
		ValueNames []string
		Values     []interface{}
	}
)

func NewStorage() *Storage {
	return &Storage{
		timelines: make(map[string]*Timeline),
	}
}

func (s *Storage) View(f func(*Storage)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	f(s)
}

func (s *Storage) Update(f func(*Storage)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	f(s)
}

func (s *Storage) GetTimeline(key string) *Timeline {
	return s.timelines[key]
}

func (s *Storage) SetTimeline(key string, t *Timeline) {
	s.timelines[key] = t
}

func (s *Storage) DeleteTimeline(key string) {
	delete(s.timelines, key)
}

func (s *Storage) Clean(expireTime int64) {
	// 此处不会修改storage, 所以加读锁即可
	count := 0
	s.View(func(s *Storage) {
		for _, t := range s.timelines {
			// 此处对t加写锁
			t.Update(func(t *Timeline) {
				t.dict = nil
				for i := range t.shards {
					s := t.shards[i]
					if s != nil && s.ts < expireTime {
						t.shards[i] = nil
						count++
					}
				}
			})
		}
	})
	logger.Infof("clean %d expired shards", count)
}

func (s *Storage) Start() {
	go func() {
		timer := time.NewTimer(time.Minute)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				ms := util.CurrentMS() - 5*60_000
				s.Clean(ms)
				timer.Reset(time.Minute)
			}
		}
	}()
}

func (s *Storage) Stop() {
}

func NewTimeline(key string, interval, capacity int64) *Timeline {
	return &Timeline{
		Key:      key,
		interval: interval,
		capacity: capacity,
	}
}

func (t *Timeline) View(f func(*Timeline)) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	f(t)
}

func (t *Timeline) Update(f func(*Timeline)) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	f(t)
}

func (t *Timeline) Lock() {
	t.mutex.Lock()
}

func (t *Timeline) Unlock() {
	t.mutex.Unlock()
}

func (t *Timeline) GetOrCreateShard(ts int64) *Shard {
	shard := t.GetShard(ts)
	if shard == nil {
		shard = t.CreateShard(ts)
	}
	return shard
}

func (t *Timeline) GetShard(ts int64) *Shard {
	if t.shards == nil {
		return nil
	}
	no := ts / t.interval
	index := no % t.capacity
	s := t.shards[index]
	if s == nil {
		return nil
	}
	if s.no != no {
		return nil
	}
	return s
}

func (t *Timeline) CreateShard(ts int64) *Shard {
	if t.shards == nil {
		t.shards = make([]*Shard, t.capacity)
	}
	no := ts / t.interval
	index := no % t.capacity
	s := &Shard{
		ts: ts,
		no: no,
	}
	t.shards[index] = s
	return s
}

func (t *Timeline) AddRef(c int) int {
	t.refCount += c
	return t.refCount
}

func (t *Timeline) RefCount() int {
	return t.refCount
}

func (s *Shard) GetPoint(key string) *Point {
	if s.points == nil {
		return nil
	}
	return s.points[key]
}

func (s *Shard) SetPoint(key string, p *Point) {
	if s.points == nil {
		s.points = make(map[string]*Point)
	}
	s.points[key] = p
}

func (s *Shard) InternalGetAllPoints() map[string]*Point {
	return s.points
}

func (s *Shard) PointCount() int {
	return len(s.points)
}
