package executor

type (
	// 子消费者, 是一种实际的日志处理
	SubConsumer interface {
		setParent(parent *Consumer)
		Update(f func())
		ProcessGroup(iw *inputWrapper, ctx *LogContext, maxTs *int64)
		Emit(expectedTs int64)
		init()
	}
)
