package collectconfig

type (
	LogAnalysisConf struct {
		// patterns to match, will be visited in order, break when first match
		Patterns []*LogAnalysisPatternConf `json:"patterns"`
		// max Count of generated unknown patterns, defaults to 64
		MaxUnknownPatterns int `json:"maxUnknownPatterns"`
		// truncate log if length(bytes) exceed MaxLogLength, defaults to 300
		MaxLogLength int `json:"maxLogLength"`
	}
	LogAnalysisPatternConf struct {
		// pattern name
		Name string `json:"name"`
		// 'where' predicate of pattern
		Where *Where `json:"where"`
	}
)
