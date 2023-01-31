package strategy

import (
	"math/rand"
	"time"
)

type BackOff struct {
	options *strategyOption

	curBackOffTime int
	proportion     float64 // 结果的左右浮动区间
}

func NewBackOffStrategy(proportion float64, options ...OptionFunc) *BackOff {
	opt := defaultOptions
	for _, o := range options {
		o(&opt)
	}

	return &BackOff{
		options:        &opt,
		curBackOffTime: opt.initTime,
		proportion:     proportion,
	}
}

func (bs *BackOff) GetNext() int {
	curBackOffTime := bs.curBackOffTime
	nextBackOffTime := curBackOffTime
	if bs.curBackOffTime < bs.options.maxBackoffTime {
		bs.curBackOffTime *= bs.options.indexFactor
	}

	if curBackOffTime >= bs.options.maxBackoffTime {
		nextBackOffTime = bs.options.maxBackoffTime
	}

	rand.Seed(time.Now().UnixNano())
	next := 1000 * nextBackOffTime
	timeRand := float64(next) + float64(next-rand.Intn(2*next))*bs.proportion

	return int(timeRand)
}

func (bs *BackOff) Reset() {
	bs.curBackOffTime = bs.options.initTime
}
