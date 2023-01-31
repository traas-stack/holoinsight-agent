package loganalysis

const (
	DefaultLogPatterns    = 64
	DefaultMaxSourceCount = 60
	DefaultMaxLoLength    = 300
)

type (
	Analyzer struct {
		logs            []*AnalyzedLog
		maxLogLength    int
		maxPatternCount int
	}
	AnalyzedLog struct {
		Parts       []*LAPart     `json:"parts,omitempty"`
		Sample      string        `json:"sample,omitempty"`
		Count       int           `json:"count,omitempty"`
		SourceWords []*SourceWord `json:"sourceWords,omitempty"`
		sources     map[string]int
	}
	SourceWord struct {
		Source string `json:"source"`
		Count  int    `json:"count"`
	}
	Result struct {
		// 错误行数
		Count int `json:"count,omitempty"`
		// 错误日志分析
		Unknown *Unknown `json:"unknown,omitempty"`
		// 已知模式匹配数量
		Known *Known `json:"known,omitempty"`
	}
	Known struct {
		Patterns []*KnownPatternStat `json:"patterns,omitempty"`
	}
	KnownPatternStat struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	Unknown struct {
		AnalyzedLogs []*AnalyzedLog `json:"analyzedLogs,omitempty"`
	}
)

func NewAnalyzer(maxLogLength, maxPatternCount int) *Analyzer {
	if maxLogLength <= 0 {
		maxLogLength = DefaultMaxLoLength
	}
	if maxPatternCount <= 0 {
		maxPatternCount = DefaultLogPatterns
	}
	return &Analyzer{
		maxLogLength:    maxLogLength,
		maxPatternCount: maxPatternCount,
	}
}

func newErrorLog(parts []*LAPart, sample string) *AnalyzedLog {
	el := &AnalyzedLog{Parts: parts, Sample: sample}
	el.mergeKeywords(parts)
	return el
}

func (el *AnalyzedLog) mergeKeywords(parts []*LAPart) {
	el.Count++
	if el.sources == nil {
		el.sources = make(map[string]int)
	}

	for _, p := range parts {
		if p.Source {
			if _, ok := el.sources[p.Content]; ok {
				el.sources[p.Content]++
			} else {
				if len(el.sources) > DefaultMaxSourceCount {
					continue
				} else {
					el.sources[p.Content] = 1
				}
			}
		}
	}
}

func (a *Analyzer) Analyze(log string) {
	if len(log) > a.maxLogLength {
		log = log[:a.maxLogLength]
	}
	parts := dissembleParts(log)
	for _, ea := range a.logs {
		if isSimilar(parts, ea.Parts) {
			ea.mergeKeywords(parts)
			return
		}
	}

	if len(a.logs) < a.maxPatternCount {
		errorLog := newErrorLog(parts, log)
		a.logs = append(a.logs, errorLog)
	}
}

func (a *Analyzer) AnalyzedLogs() []*AnalyzedLog {
	for _, log := range a.logs {
		log.SourceWords = make([]*SourceWord, 0, len(log.sources))
		for source, count := range log.sources {
			log.SourceWords = append(log.SourceWords, &SourceWord{
				Source: source,
				Count:  count,
			})
		}
	}
	return a.logs
}

func (a *Analyzer) Clear() {
	a.logs = nil
}
