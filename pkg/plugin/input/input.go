package input

import "github.com/traas-stack/holoinsight-agent/pkg/model"

type (
	CollectContext struct{}

	// TODO 约定线程模型
	// 一般情况下, 外界必须保证对同一个input的调用是串行的
	Input interface {
		// Type() string
		// TODO need collect context ?
		// Collect(ctx)
		Collect(ctx *CollectContext) ([]*model.DetailData, error)

		model.Module
	}
	StatefulInput interface {
		Input
		SerializeState() ([]byte, error)
		DeserializeState([]byte) error
	}
	BaseInput struct{}
)

func (*BaseInput) Start() {
}
func (*BaseInput) Stop() {
}
