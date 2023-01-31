package stats

import (
	"testing"
	"time"
)

func TestSpm(t *testing.T) {
	sm := NewSpmStatsManager(time.Second*4, func(table string, result bool, count, cost uint64) {
		t.Logf("[spm_%s] table=%s result=%+v count=%d cost=%dms", "serviceA", table, result, count, cost)
	})
	s := sm.Spawn("table1")
	time.Sleep(time.Second)
	s.Done(true)

	s = sm.Spawn("table1")
	time.Sleep(time.Second * 2)
	s.Done(true)

	s = sm.Spawn("table1")
	time.Sleep(time.Second)
	s.Done(false)

	s = sm.Spawn("table1")
	time.Sleep(time.Second)
	s.Reset()
	time.Sleep(time.Second)
	s.Done(false)

	time.Sleep(time.Second * 10)
}

func TestStats(t *testing.T) {
	mgr := NewStatsManager(time.Second, func(metrics map[string]uint64) {
		t.Log(metrics)
	})
	mgr.AddAvg("cost", 1)
	mgr.AddAvg("cost", 200)

	time.Sleep(time.Second * 60)
}

func TestTableStats(t *testing.T) {
	mgr := NewTableStatsManager(time.Second, func(metrics map[string]map[string]uint64) {
		t.Log(metrics)
	})

	mgr.Add("table1", "a", 1)
	mgr.Add("table1", "b", 2)
	time.Sleep(time.Second * 60)
}
