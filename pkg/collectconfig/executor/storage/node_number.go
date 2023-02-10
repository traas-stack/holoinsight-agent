package storage

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/agg"
	"math"
)

type (
	AggNumberDataNode struct {
		Value float64
		Count int32
		Agg   agg.AggType
	}
)

func (n *AggNumberDataNode) MergeHll(str string) {
	panic("AggNumberDataNode doesn't support MergeHll")
}

func NewAggNumberDataNode(aggType agg.AggType) DataNode {
	return &AggNumberDataNode{
		Agg: aggType,
	}
}

func (n *AggNumberDataNode) String() string {
	return fmt.Sprintf("[%d,%f]", n.Count, n.Value)
}

func (n *AggNumberDataNode) AddCount() {
	n.Count++
	n.Value = float64(n.Count)
}

func (n *AggNumberDataNode) MergeNumber(f float64) {
	n.Count++
	switch n.Agg {
	case agg.AggSum:
		n.Value += f
	case agg.AggAvg:
		n.Value += f
	case agg.AggMin:
		if n.Count == 1 {
			n.Value = f
		} else {
			// TODO archMin 是什么? 性能测试
			n.Value = math.Min(n.Value, f)
		}
	case agg.AggMax:
		n.Value += f
		if n.Count == 1 {
			n.Value = f
		} else {
			n.Value = math.Max(n.Value, f)
		}
	case agg.AggCount:
		// AggCount是特殊的其value总是等于其count
		// 理论上代码不会执行到这里 而是会调用 addCount
		n.Value = float64(n.Count)
	default:
		break
	}
}
