package stats

import (
	"sync"
	"time"

	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

type (
	TableStatsManager struct {
		defaultReporter TableStatsExporter
		tableMetrics    map[string]*tableEntry
		metricsChan     chan *metricItem

		metricItemPool sync.Pool
	}
	StatsManager struct {
		*TableStatsManager
	}

	tableEntry struct {
		metrics map[string]metricInfo

		isActive bool
	}

	metricItem struct {
		table string

		key   string
		value uint64

		mode calMode
		cb   func() uint64
	}
	metricInfo struct {
		v    uint64
		n    int // v是多少个数的和
		mode calMode
		cb   func() uint64
	}
	calMode            uint8
	TableStatsExporter func(metrics map[string]map[string]uint64)
	StatsExporter      func(metrics map[string]uint64)
)

const (
	defaultChanSize  = 65536
	defaultTableName = "defaultTable"
)

const (
	calModeSum calMode = iota
	calModeAvg
	calModeGauge
	calModeMax
)

func NewTableStatsManager(reportDura time.Duration, r TableStatsExporter) *TableStatsManager {
	ret := &TableStatsManager{
		defaultReporter: r,
		tableMetrics:    make(map[string]*tableEntry),
		metricsChan:     make(chan *metricItem, defaultChanSize),
		metricItemPool:  sync.Pool{New: func() interface{} { return &metricItem{} }},
	}
	ret.statsLoop(reportDura)

	return ret
}

func NewStatsManager(reportDura time.Duration, r StatsExporter) *StatsManager {
	return &StatsManager{
		NewTableStatsManager(reportDura, func(metrics map[string]map[string]uint64) {
			r(metrics[defaultTableName])
		}),
	}
}

func (tsm *TableStatsManager) statsLoop(reportDura time.Duration) {
	util.GoWithRecover(func() {
		tk := time.NewTicker(reportDura)
		defer tk.Stop()

		clearTk := time.NewTicker(time.Hour)
		defer clearTk.Stop()

		for {
			select {
			case <-clearTk.C:
				var deleteKeys []string
				for t, m := range tsm.tableMetrics {
					if !m.isActive {
						deleteKeys = append(deleteKeys, t)
					}
				}
				if len(deleteKeys) == len(tsm.tableMetrics) {
					tsm.tableMetrics = make(map[string]*tableEntry)
				} else {
					for _, k := range deleteKeys {
						delete(tsm.tableMetrics, k)
					}
				}
			case <-tk.C:
				msm := make(map[string]map[string]uint64)
				for t, m := range tsm.tableMetrics {
					if !m.isActive {
						continue
					}
					if tsm.defaultReporter != nil {
						ms := make(map[string]uint64, len(m.metrics))
						for k, v := range m.metrics {
							switch v.mode {
							case calModeGauge:
								if v.cb != nil {
									v.v = v.cb() // cover
								}
							case calModeAvg:
								v.v = v.v / uint64(v.n)
							default:
							}
							ms[k] = v.v
						}
						msm[t] = ms
					}
					for k, v := range m.metrics {
						if v.mode != calModeGauge {
							delete(m.metrics, k)
						}
					}
					m.isActive = false
				}
				if tsm.defaultReporter != nil {
					tsm.defaultReporter(msm)
				}
			case x := <-tsm.metricsChan:
				if _, exist := tsm.tableMetrics[x.table]; !exist {
					tsm.tableMetrics[x.table] = &tableEntry{metrics: map[string]metricInfo{}}
				}

				tbl := tsm.tableMetrics[x.table]
				mi := tbl.metrics[x.key]
				mi.v += x.value
				mi.n += 1
				mi.mode = x.mode
				mi.cb = x.cb
				tbl.metrics[x.key] = mi

				tbl.isActive = true
				tsm.metricItemPool.Put(x)
			}
		}
	}, func(_ interface{}) {
		tsm.statsLoop(reportDura)
	})
}

func (tsm *TableStatsManager) newMetricItem(table string, key string, value uint64) *metricItem {
	m := tsm.metricItemPool.Get().(*metricItem)
	m.table = table
	m.key = key
	m.value = value
	m.mode = calModeSum
	m.cb = nil
	return m
}

func (tsm *TableStatsManager) add(m *metricItem) {
	select {
	case tsm.metricsChan <- m:
	default:
	}
}

func (tsm *TableStatsManager) Add(table string, key string, value uint64) {
	if value <= 0 {
		return
	}
	tsm.add(tsm.newMetricItem(table, key, value))
}

func (tsm *TableStatsManager) AddAvg(table string, key string, value uint64) {
	if value <= 0 {
		return
	}
	m := tsm.newMetricItem(table, key, value)
	m.mode = calModeAvg
	tsm.add(m)
}

func (sm *StatsManager) Add(key string, value uint64) {
	if value <= 0 {
		return
	}
	sm.add(sm.newMetricItem(defaultTableName, key, value))
}

func (sm *StatsManager) AddAvg(key string, value uint64) {
	if value <= 0 {
		return
	}
	m := sm.newMetricItem(defaultTableName, key, value)
	m.mode = calModeAvg
	sm.add(m)
}

func (sm *StatsManager) AddGauge(key string, cb func() uint64) {
	m := sm.newMetricItem(defaultTableName, key, 0)
	m.mode = calModeGauge
	m.cb = cb
	sm.add(m)
}
