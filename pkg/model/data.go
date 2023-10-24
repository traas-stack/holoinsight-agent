/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/traas-stack/holoinsight-agent/pkg/util"
)

type (
	Metric struct {
		Name      string            `json:"name"`
		Tags      map[string]string `json:"tags"`
		Timestamp int64             `json:"timestamp"`
		Value     float64           `json:"value"`
	}
	DetailData struct {
		Timestamp int64
		// 明确是tags
		Tags map[string]string
		// TODO 似乎有一部分case values也是string 此时要和tags区分开
		Values map[string]interface{}
	}

	// header
	Schema struct {
		StringNames []string
		MetricNames []string
	}

	Module interface {
		Start()

		Stop()
	}

	Addr struct {
		Ip   string
		Port int
	}
	Table struct {
		Name   string  `json:"name"`
		Header *Header `json:"header"`
		Rows   []*Row  `json:"rows"`
	}
	Header struct {
		TagKeys   []string `json:"tagKeys"`
		FieldKeys []string `json:"fieldKeys"`
	}
	Row struct {
		Timestamp   int64     `json:"timestamp"`
		TagValues   []string  `json:"tagValues"`
		FieldValues []float64 `json:"fieldValues"`
	}
)

func (a Addr) String() string {
	return fmt.Sprintf("%s:%d", a.Ip, a.Port)
}

func NewDetailData() *DetailData {
	return &DetailData{
		Timestamp: util.CurrentMS(),
		Tags:      make(map[string]string),
		Values:    make(map[string]interface{}),
	}
}

func (dd *DetailData) WithTag(k string, v string) *DetailData {
	dd.Tags[k] = v
	return dd
}

func (dd *DetailData) WithTags(tags map[string]string) *DetailData {
	dd.Tags = tags
	return dd
}

func (dd *DetailData) WithValue(k string, v interface{}) *DetailData {
	dd.Values[k] = v
	return dd
}

func (dd *DetailData) WithValues(values map[string]interface{}) *DetailData {
	dd.Values = values
	return dd
}

func MakeDetailDataSlice(dd *DetailData, dds ...*DetailData) []*DetailData {
	r := make([]*DetailData, 0, len(dds)+1)
	r = append(r, dd)
	for _, v := range dds {
		r = append(r, v)
	}
	return r
}

func (m *Metric) String() string {
	return fmt.Sprintf("name=[%s] ts=[%d] tags=%v value=[%f]", m.Name, m.Timestamp, m.Tags, m.Value)
}

// BuildMetricKey builds the key for the given Metric object.
// So it can be stored in map[string]float64.
func BuildMetricKey(m *Metric) string {
	sb := strings.Builder{}

	sb.WriteString(m.Name)

	keys := make([]string, 0, len(m.Tags))
	for k := range m.Tags {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, key := range keys {
		sb.WriteByte(',')
		sb.WriteString(key)
		sb.WriteByte('=')
		sb.WriteString(m.Tags[key])
	}

	return sb.String()
}
