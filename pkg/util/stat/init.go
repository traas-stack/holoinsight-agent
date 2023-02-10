package stat

import (
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"strconv"
	"strings"
)

var DefaultManager *Manager
var DefaultManager1S *Manager

type (
	internalLoggerPrinter struct {
	}
)

func (ignored *internalLoggerPrinter) Print(st StatEvent) {
	sb := strings.Builder{}
	for _, c := range st.CounterItems {

		for key, values := range c.Data {
			if len(values) == 0 {
				continue
			}

			sb.Reset()

			sb.WriteString(c.Name)
			if len(key) > 0 {
				sb.WriteByte(',')
				sb.WriteString(key)
			}

			for _, value := range values {
				sb.WriteByte(',')
				sb.WriteString(strconv.FormatInt(value, 10))
			}
			logger.Stat(sb.String())
		}
	}

	for _, g := range st.GaugeItems {
		for _, subItem := range g.SubItems {
			if len(subItem.Values) == 0 {
				continue
			}

			sb.Reset()

			sb.WriteString(g.Name)

			for _, key := range subItem.Keys {
				sb.WriteByte(',')
				sb.WriteString(key)
			}

			for _, value := range subItem.Values {
				sb.WriteByte(',')
				sb.WriteString(strconv.FormatInt(value, 10))
			}
			logger.Stat(sb.String())
		}
	}

}

func init() {
	DefaultManager = NewStatManager(60, new(internalLoggerPrinter))
	DefaultManager.Start()
	DefaultManager1S = NewStatManager(1, new(internalLoggerPrinter))
	DefaultManager1S.Start()
}

func SetupStat() {
	// 该模块在运行时依赖 logger.Stat
}
