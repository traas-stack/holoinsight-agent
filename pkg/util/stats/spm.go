package stats

import (
	"sync"
	"time"
)

type (
	SpmStatsManager struct {
		p sync.Pool

		success, fail *TableStatsManager
	}

	SpmOnce struct {
		factory *SpmStatsManager

		table string
		start time.Time
		done  bool
	}
	SpmExporter func(table string, result bool, count, cost uint64)
)

func NewSpmStatsManager(d time.Duration, r SpmExporter) *SpmStatsManager {
	return &SpmStatsManager{
		p: sync.Pool{
			New: func() interface{} {
				return &SpmOnce{}
			},
		},
		success: NewTableStatsManager(d, func(metrics map[string]map[string]uint64) {
			for k, v := range metrics {
				r(k, true, v["count"], v["cost"])
			}
		}),
		fail: NewTableStatsManager(d, func(metrics map[string]map[string]uint64) {
			for k, v := range metrics {
				r(k, false, v["count"], v["cost"])
			}
		}),
	}
}

func (ssm *SpmStatsManager) Spawn(table ...string) *SpmOnce {
	r := ssm.p.Get().(*SpmOnce)
	// init
	r.done = false
	r.factory = ssm
	if len(table) != 0 {
		r.table = table[0]
	}
	r.start = time.Now()
	return r
}

func (so *SpmOnce) Reset() {
	so.done = false
	so.start = time.Now()
}

func (so *SpmOnce) Done(result bool) {
	if so.done {
		return
	}

	so.done = true
	cost := time.Now().Sub(so.start)
	ssm := so.factory

	m := ssm.fail
	if result {
		m = ssm.success
	}

	m.Add(so.table, "count", 1)
	// 这里统计对应count数的总耗时
	m.Add(so.table, "cost", uint64(cost.Nanoseconds()/1e6))

	ssm.p.Put(so)
}
