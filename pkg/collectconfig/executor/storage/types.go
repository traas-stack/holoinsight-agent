/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package storage

type (
	// TODO 这里似乎将 agg 的落沉淀到底层的datanode上
	// 这是我不太想的, 但如果不这么做, 底层就要存储 interface{}
	DataNode interface {
		AddCount()
		MergeNumber(float64)
		MergeHll(string)
	}
)
