package common

import "time"

type (
	// 系统指标采集器
	// 这里并没有像 xflush-agent 一样 是很多细粒度采集方法, 而是只有一个很简单的 start
	// 因此如果实现不一样(cadvisor/asar) 那么总体的执行流程就很不一样
	// 不像 asar/vsar 只有细节的差异
	SysCollector interface {
		Name() string
		Start()
		Stop()
	}
	SysCollectorProvider func(suffix string, interval time.Duration) (SysCollector, error)
)
