/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package stat

import (
	"github.com/traas-stack/holoinsight-agent/pkg/util"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var V_1 = []int64{1}
var V_1_0 = []int64{1, 0}
var V_0_1 = []int64{0, 1}
var V_1_1 = []int64{1, 1}

type (
	Logger  func(string)
	Manager struct {
		intervalSecond int64
		mutex          sync.Mutex
		adders         []*Adder
		adderMap       map[string]*Adder
		gaugeMap       map[string]Gauger
		status         int32
		period         time.Time
		printer        Printer
	}

	Printer interface {
		Print(st StatEvent)
	}

	Adder struct {
		mutex   sync.Mutex
		name    string
		manager *Manager
		// 这里遇到问题了 需要以 []string为key, 但不支持...
		data map[string][]int64
	}

	Gauger func() []GaugeSubItem

	Bind struct {
		a         *Adder
		joinedKey string
	}

	StatEvent struct {
		Period       time.Time
		Now          time.Time
		CounterItems []CounterItem
		GaugeItems   []GaugeItem
	}

	CounterItem struct {
		Name string
		Data map[string][]int64
	}

	GaugeItem struct {
		Name     string
		SubItems []GaugeSubItem
	}

	GaugeSubItem struct {
		Keys   []string
		Values []int64
	}
)

func NewStatManager(intervalSecond int, printer Printer) *Manager {
	return &Manager{
		intervalSecond: int64(intervalSecond),
		adderMap:       make(map[string]*Adder),
		gaugeMap:       make(map[string]Gauger),
		printer:        printer,
	}
}

func (m *Manager) Start() {
	go m.run()
}

func (m *Manager) Stop() {
	// TODO 不能关闭ch 我们无法原子 除非加锁
	atomic.StoreInt32(&m.status, 1)
}

func (m *Manager) Counter(name string) *Adder {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if a, ok := m.adderMap[name]; ok {
		return a
	}

	a := &Adder{
		name:    name,
		manager: m,
		data:    make(map[string][]int64),
	}
	m.adderMap[name] = a
	m.adders = append(m.adders, a)
	return a
}

func (m *Manager) Gauge(name string, gauger Gauger) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.gaugeMap[name]; ok {
		return false
	}
	m.gaugeMap[name] = gauger
	return true
}

func (m *Manager) IsStopped() bool {
	return atomic.LoadInt32(&m.status) == 1
}

func (m *Manager) run() {
	timer, _ := util.NewAlignedTimer(time.Duration(m.intervalSecond)*time.Second, 0, true, false)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			func() {
				m.mutex.Lock()
				defer m.mutex.Unlock()

				if m.IsStopped() {
					return
				}

				now := time.Now()

				counterItems := make([]CounterItem, len(m.adders))
				for i, a := range m.adders {
					counterItems[i] = CounterItem{
						Name: a.name,
						Data: a.getAndClear(),
					}
				}

				gaugeItems := make([]GaugeItem, 0, len(m.gaugeMap))
				for name, gauger := range m.gaugeMap {
					gaugeItems = append(gaugeItems, GaugeItem{
						Name:     name,
						SubItems: gauger(),
					})
				}

				st := StatEvent{
					Period:       m.period,
					Now:          now,
					CounterItems: counterItems,
					GaugeItems:   gaugeItems,
				}

				m.print(st)

				timer.Next()
			}()
		}
	}
}

func (m *Manager) print(st StatEvent) {
	m.printer.Print(st)
}

func (a *Adder) Add(keys []string, values []int64) {
	a.add(joinKey(keys), values)
}

// keys本身必须不能包含','
func (a *Adder) Bind(keys ...string) *Bind {
	return &Bind{
		a:         a,
		joinedKey: joinKey(keys),
	}
}

func (b *Bind) Add(values ...int64) {
	b.a.add(b.joinedKey, values)
}

func (a *Adder) add(key string, values []int64) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	var acc []int64
	var ok bool
	if acc, ok = a.data[key]; ok {
		if len(acc) != len(values) {
			return
		}
	} else {
		acc = make([]int64, len(values))
		a.data[key] = acc
		// should we need to zero the slice?
	}
	for i := 0; i < len(acc); i++ {
		acc[i] += values[i]
	}
}

func (a *Adder) getAndClear() map[string][]int64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	data := a.data
	a.data = make(map[string][]int64)
	return data
}

func joinKey(keys []string) string {
	return strings.Join(keys, ",")
}
