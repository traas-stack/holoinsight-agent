package output

import (
	"errors"

	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

const (
	ConsoleType = "console"
)

type (
	Config  interface{}
	Factory func(Config) (Output, error)
)

var factories = make(map[string]Factory)

func Register(outputType string, factory Factory) {
	if _, exist := factories[outputType]; exist {
		logger.Warnf("[plugin] register output factory %+v already exist, cover it", outputType)
	}
	factories[outputType] = factory
}

// TODO 有一些output实例要复用的, 我们可以在最底层的output上再包一个RefOutput
// 第一次创建时触发底层output创建, 然后持有住, 记录refCount=1
// 之后再次创建就复用output, refCount++
// 销毁之后refCount==0就释放底层output
func Parse(outputType string, config Config) (Output, error) {
	// TODO 临时的额
	if outputType == "" {
		outputType = "gateway"
	}
	if f, ok := factories[outputType]; ok {
		return f(config)
	} else {
		return nil, errors.New("unsupported output type " + outputType)
	}
}
