/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package loganalysis

const (
	DefaultLogPatterns    = 64
	DefaultMaxSourceCount = 60
	DefaultMaxLoLength    = 300
)

type (
	// Analyzer
	// This struct needs to be encoded by gob. So all fields are public.
	Analyzer struct {
		Logs            []*AnalyzedLog
		MaxLogLength    int
		MaxPatternCount int
	}
	AnalyzedLog struct {
		Parts       []*LAPart      `json:"parts,omitempty"`
		Sample      string         `json:"sample,omitempty"`
		Count       int            `json:"count,omitempty"`
		SourceWords []*SourceWord  `json:"sourceWords,omitempty"`
		Sources     map[string]int `json:"-"`
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
		MaxLogLength:    maxLogLength,
		MaxPatternCount: maxPatternCount,
	}
}

func newErrorLog(parts []*LAPart, sample string) *AnalyzedLog {
	el := &AnalyzedLog{Parts: parts, Sample: sample}
	el.mergeKeywords(parts)
	return el
}

func (el *AnalyzedLog) mergeKeywords(parts []*LAPart) {
	el.Count++
	if el.Sources == nil {
		el.Sources = make(map[string]int)
	}

	for _, p := range parts {
		if p.Source {
			if _, ok := el.Sources[p.Content]; ok {
				el.Sources[p.Content]++
			} else {
				if len(el.Sources) > DefaultMaxSourceCount {
					continue
				} else {
					el.Sources[p.Content] = 1
				}
			}
		}
	}
}

func (a *Analyzer) Analyze(log string) {
	if len(log) > a.MaxLogLength {
		log = log[:a.MaxLogLength]
	}
	parts := dissembleParts(log)
	for _, ea := range a.Logs {
		if isSimilar(parts, ea.Parts) {
			ea.mergeKeywords(parts)
			return
		}
	}

	if len(a.Logs) < a.MaxPatternCount {
		errorLog := newErrorLog(parts, log)
		a.Logs = append(a.Logs, errorLog)
	}
}

func (a *Analyzer) AnalyzedLogs() []*AnalyzedLog {
	for _, log := range a.Logs {
		log.SourceWords = make([]*SourceWord, 0, len(log.Sources))
		for source, count := range log.Sources {
			log.SourceWords = append(log.SourceWords, &SourceWord{
				Source: source,
				Count:  count,
			})
		}
	}
	return a.Logs
}

func (a *Analyzer) Clear() {
	a.Logs = nil
}
