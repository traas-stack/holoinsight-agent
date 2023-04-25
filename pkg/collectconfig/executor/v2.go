/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

type (
	LogSource interface {
		Start()
		Stop()
	}
	// 拉模式的logsource
	PullLogSource interface {
		LogSource
	}
	LogConsumer        interface{}
	LogConsumerManager struct {
		consumers map[string]LogConsumer
	}
	LogTaskPipeline struct {
		Source          LogSource
		ConsumerManager LogConsumerManager
	}
)
