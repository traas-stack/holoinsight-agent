/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package storage

type (
	HllDataNode struct {
	}
)

func NewHllDataNode() DataNode {
	return &HllDataNode{}
}

func (h *HllDataNode) AddCount() {
	panic("HllDataNode doesn't support AddCount")
}

func (h *HllDataNode) MergeNumber(f float64) {
	panic("HllDataNode doesn't support MergeNumber")
}

func (h *HllDataNode) MergeHll(str string) {
	// TODO 待实现
}
