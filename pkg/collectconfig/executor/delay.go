package executor

type (
	DelayCalculator struct{}
)

func NewDelayCalculator() *DelayCalculator {
	return &DelayCalculator{}
}
