package executor

type (
	// 子消费者, 是一种实际的日志处理
	SubConsumer interface {
		setParent(parent *Consumer)
		Update(f func())
		ProcessGroup(iw *inputWrapper, ctx *LogContext, maxTs *int64)
		// Emit emits data with timestamp equals to expectedTs
		// Returns true if data is not empty
		Emit(expectedTs int64) bool
		init()
	}
)
