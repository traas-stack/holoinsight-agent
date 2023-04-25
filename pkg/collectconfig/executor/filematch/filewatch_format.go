/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

type (
	FormatFileMatcher struct {
		pattern string
	}
)

func (m *FormatFileMatcher) IsDynamicMultiFiles() bool {
	return false
}

func NewFormatFileMatcher(pattern string) *FormatFileMatcher {
	return &FormatFileMatcher{
		pattern: pattern,
	}
}

func (m *FormatFileMatcher) Find() ([]FatPath, int, error) {
	return []FatPath{
		{
			Path: stdFilePathFormatter.Format(m.pattern),
		},
	}, 0, nil
}
