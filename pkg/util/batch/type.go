/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

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
