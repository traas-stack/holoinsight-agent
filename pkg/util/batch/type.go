package batch

type (
	Processor interface {
		Put(interface{})
		TryPut(interface{}) bool

		Run()
		Shutdown()

		Num() int

		AdjustBatchMaxSize(int)
		AdjustBatchMaxWait(duration int)

		flush([]interface{})
	}

	Consumer interface {
		// must be reentrancy
		Consume([]interface{})
	}

	KeyBatchProcessor interface {
		TryPut(string, interface{}) bool

		Run()
		Shutdown()

		KeyNum() int
		BufferNum() int

		AdjustBatchMaxSize(int)
		AdjustBatchMaxWait(duration int)

		flush(string, []interface{})
	}

	KeyConsumer interface {
		// must be reentrancy
		Consume(string, []interface{})
	}
)
