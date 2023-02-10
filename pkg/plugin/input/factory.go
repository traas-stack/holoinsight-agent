package input

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/traas-stack/holoinsight-agent/pkg/logger"
)

type (
	Factory func(Config) (Input, error)
	// Config是抽象的, 需要根据各个type而定
	Config interface{}
)

var factories = make(map[string]Factory)

func Register(inputType string, factory Factory) {
	if _, exist := factories[inputType]; exist {
		logger.Warnf("[plugin] register input factory %+v already exist, cover it", inputType)
	}
	factories[inputType] = factory
}

// 理论上不会有关于本机的多个相同类型的input任务
func Parse(inputType string, config Config) (_ Input, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			logger.Errorf("send registry heartbeat errors, stack: %v\n%s", r, buf)

			retErr = fmt.Errorf("parse error %+v", r)
		}
	}()

	if f, ok := factories[inputType]; ok {
		return f(config)
	}
	return nil, errors.New("unsupported input type " + inputType)
}
