package agent

import (
	"context"
	regpb "github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"sync"
	"time"

	"github.com/TRaaSStack/holoinsight-agent/pkg/agent/agentmeta"
	"github.com/TRaaSStack/holoinsight-agent/pkg/logger"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util/eventcollector"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util/stats"
	"go.uber.org/zap"
)

type (
	statsCache struct {
		timestamp int64
		cache     map[string]map[string]uint64
		cacheMu   sync.RWMutex
	}
	eventsCache struct {
		cache   []*eventcollector.Event
		cacheMu sync.RWMutex
	}
)

var (
	defaultRunInfoReportDuration = time.Minute
	// 这里全局变量的存在是因为桥接单例，有点丑
	defaultStatsCache, defaultEventCache = &statsCache{}, &eventsCache{}
	// 这里固定死了周期为分钟，如果需要其他周期，可另外new一个manager
	defaultTableStatsManager = stats.NewTableStatsManager(defaultRunInfoReportDuration, func(metrics map[string]map[string]uint64) {
		// log
		logger.Infoz("[run_stats]", zap.Any("metrics", metrics))
		// push
		defaultStatsCache.cacheMu.Lock()
		defaultStatsCache.cache = metrics
		defaultStatsCache.timestamp = util.CurrentMS()
		defaultStatsCache.cacheMu.Unlock()
	})

	defaultEventCollector = eventcollector.New(defaultRunInfoReportDuration, func(events []*eventcollector.Event) {
		// log
		logger.Infoz("[run_events]", zap.Any("events", events))
		// push
		defaultEventCache.cacheMu.Lock()
		defaultEventCache.cache = events
		defaultEventCache.cacheMu.Unlock()
	})
)

// for collect task
// 具体某个采集配置的统计信息
func AddTableStat(table string, key string, value uint64) {
	defaultTableStatsManager.Add(table, key, value)
}

// for agent self
// 自身运行状态除了cpu，mem，还有些致命错误或重要统计，比如数据网关的push质量
func AddStat(key string, value uint64) {
	defaultTableStatsManager.Add("agentRunStats", key, value)
}

// for event
func AddEvent(key string, msg string) {
	defaultEventCollector.Collect(key, msg)
}

func (m *Manager) reportRunInfo() {
	if m.reportInfoDisable {
		return
	}

	m.reportRunStats()
	m.reportRunEvents()
}

func (m *Manager) reportRunStats() {
	defaultStatsCache.cacheMu.RLock()
	cache := defaultStatsCache.cache
	ts := defaultStatsCache.timestamp
	defaultStatsCache.cacheMu.RUnlock()

	if len(cache) == 0 {
		return
	}

	agentRunStats := make(map[string]uint64)
	var collectConfigRunStats []*regpb.CollectConfigRunStats

	for k, v := range cache {
		switch k {
		case "agentRunStats":
			agentRunStats = v
		default:
			collectConfigRunStats = append(collectConfigRunStats, &regpb.CollectConfigRunStats{
				TableName: k,
				Stats:     v,
			})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rsp, err := m.rs.ReportRunStats(ctx, agentmeta.GetAgentId(), ts,
		agentRunStats, collectConfigRunStats)
	if err != nil {
		logger.Errorz("[run_stats] report run stats fail", zap.Error(err))
	} else {
		logger.Infoz("[run_stats] report run stats", zap.Stringer("rsp", rsp))
	}
}

func (m *Manager) reportRunEvents() {
	defaultEventCache.cacheMu.RLock()
	cache := defaultEventCache.cache
	defaultEventCache.cacheMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rsp, err := m.rs.ReportRunEvents(ctx, agentmeta.GetAgentId(), cache)
	if err != nil {
		logger.Errorz("[run_stats] report run events fail", zap.Error(err))
	} else {
		logger.Infoz("[run_stats] report run events", zap.Stringer("rsp", rsp))
	}
}
