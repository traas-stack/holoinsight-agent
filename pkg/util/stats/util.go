package stats

func SingleExporter(f func(tableX string, metricsX map[string]uint64)) TableStatsExporter {
	return func(metrics map[string]map[string]uint64) {
		for k, v := range metrics {
			f(k, v)
		}
	}
}
