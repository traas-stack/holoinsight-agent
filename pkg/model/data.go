package model

import (
	"fmt"

	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
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
		Values      map[string]interface{}
		SingleValue bool
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
