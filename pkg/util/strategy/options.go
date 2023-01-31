package strategy

type (
	strategyOption struct {
		initTime       int
		maxBackoffTime int
		indexFactor    int
	}

	OptionFunc func(*strategyOption)
)

const (
	// 初始重试时间1s
	defaultInitTime = 1
	// 默认最大重试时间为60s
	defaultMaxBackoffTime = 60
	// 退避因子 2
	defaultIndexFactor = 2
)

var (
	defaultOptions = strategyOption{
		initTime:       defaultInitTime,
		maxBackoffTime: defaultMaxBackoffTime,
		indexFactor:    defaultIndexFactor,
	}
)

func WithInitTime(d int) OptionFunc {
	return func(option *strategyOption) {
		option.initTime = d
	}
}

func WithMaxBackoffTime(d int) OptionFunc {
	return func(option *strategyOption) {
		option.maxBackoffTime = d
	}
}

func WithIndexFactor(d int) OptionFunc {
	return func(option *strategyOption) {
		option.indexFactor = d
	}
}
