package api

type (
	// 所有新设计的插件必须非常简单, 不包含我们的业务语义, 就像 telegraf.MySQL 插件一样可以被我直接拿来运行
	// 上层我们必须要再包装一个插件, 是含有我们业务语义的.
	Input interface {
		// TODO 加这个的作用是 方便上层将来可以转成 多值 字段
		// Prefix()string
		Collect(Accumulator) error
	}
	CollectContext struct {
		AlignedTimestamp int64
	}
	InputExt interface {
		UpdateFrom(interface{})
		DebugInfo() map[string]interface{}
	}
)

const (
	NetworkModeAgent = "AGENT"
	NetworkModePod   = "POD"
)
