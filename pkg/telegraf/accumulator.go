package telegraf

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/accumulator"
	"github.com/TRaaSStack/holoinsight-agent/pkg/model"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"github.com/influxdata/telegraf"
	"github.com/spf13/cast"
	"time"
)

type (
	TelegrafAccumulatorAdapter struct {
		Separator string
		AA        accumulator.Accumulator
	}
)

func (a *TelegrafAccumulatorAdapter) AddFields(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	var timestamp int64
	if len(t) == 0 {
		timestamp = util.CurrentMS()
	} else {
		timestamp = t[0].UnixMilli()
	}
	for k, v := range fields {
		f64, err := cast.ToFloat64E(v)
		if err != nil {
			continue
		}
		var name string
		if k == "" {
			name = measurement
		} else {
			name = measurement + a.Separator + k
		}
		a.AA.AddMetric(&model.Metric{
			Name:      name,
			Tags:      tags,
			Timestamp: timestamp,
			Value:     f64,
		})
	}
}

func (a *TelegrafAccumulatorAdapter) AddGauge(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	a.AddFields(measurement, fields, tags, t...)
}

func (a *TelegrafAccumulatorAdapter) AddCounter(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	a.AddFields(measurement, fields, tags, t...)
}

func (a *TelegrafAccumulatorAdapter) AddSummary(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	a.AddFields(measurement, fields, tags, t...)
}

func (a *TelegrafAccumulatorAdapter) AddHistogram(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	a.AddFields(measurement, fields, tags, t...)
}

func (a *TelegrafAccumulatorAdapter) AddMetric(metric telegraf.Metric) {
	measurement := metric.Name()
	timestamp := metric.Time().UnixMilli()
	for k, v := range metric.Fields() {
		f64, err := cast.ToFloat64E(v)
		if err != nil {
			continue
		}
		var name string
		if k == "" {
			name = measurement
		} else {
			name = measurement + a.Separator + k
		}
		a.AA.AddMetric(&model.Metric{
			Name:      name,
			Tags:      metric.Tags(),
			Timestamp: timestamp,
			Value:     f64,
		})
	}
}

func (a *TelegrafAccumulatorAdapter) SetPrecision(precision time.Duration) {
	// noop
}

func (a *TelegrafAccumulatorAdapter) AddError(err error) {
	a.AA.AddError(err)
	// noop
}

func (a *TelegrafAccumulatorAdapter) WithTracking(maxTracked int) telegraf.TrackingAccumulator {
	return nil
}

func Adapt(aa accumulator.Accumulator, separator string) telegraf.Accumulator {
	if separator == "" {
		separator = "_"
	}
	return &TelegrafAccumulatorAdapter{AA: aa, Separator: separator}
}
