package telegraf

import (
	"github.com/influxdata/telegraf"
	tmetric "github.com/influxdata/telegraf/metric"
	"time"
)

type (
	Memory struct {
		Metrics []telegraf.Metric
	}
)

func (m *Memory) AddFields(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	if len(t) == 0 {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, time.Now(), telegraf.Untyped))
	} else {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, t[0], telegraf.Untyped))
	}
}

func (m *Memory) AddGauge(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	if len(t) == 0 {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, time.Now(), telegraf.Gauge))
	} else {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, t[0], telegraf.Gauge))
	}
}

func (m *Memory) AddCounter(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	if len(t) == 0 {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, time.Now(), telegraf.Counter))
	} else {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, t[0], telegraf.Counter))
	}
}

func (m *Memory) AddSummary(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	if len(t) == 0 {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, time.Now(), telegraf.Summary))
	} else {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, t[0], telegraf.Summary))
	}
}

func (m *Memory) AddHistogram(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	if len(t) == 0 {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, time.Now(), telegraf.Histogram))
	} else {
		m.Metrics = append(m.Metrics, tmetric.New(measurement, tags, fields, t[0], telegraf.Histogram))
	}
}

func (m *Memory) AddMetric(metric telegraf.Metric) {
	m.Metrics = append(m.Metrics, metric)
}

func (m *Memory) SetPrecision(precision time.Duration) {
}

func (m *Memory) AddError(err error) {
}

func (m *Memory) WithTracking(maxTracked int) telegraf.TrackingAccumulator {
	return nil
}
