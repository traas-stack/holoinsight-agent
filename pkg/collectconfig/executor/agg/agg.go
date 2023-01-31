package agg

import (
	"strings"
)

const (
	AggUnknown AggType = iota
	AggSum
	AggAvg
	AggMax
	AggMin
	AggCount
	AggHll
	AggLogAnalysis
)

type (
	AggType uint8
)

func GetAggType(agg string) AggType {
	switch strings.ToUpper(agg) {
	case "SUM":
		return AggSum
	case "AVG":
		return AggAvg
	case "MIN":
		return AggMin
	case "MAX":
		return AggMax
	case "COUNT":
		return AggCount
	case "HLL":
		return AggHll
	case "LOGANALYSIS":
		return AggLogAnalysis
	default:
		return AggUnknown
	}
}
