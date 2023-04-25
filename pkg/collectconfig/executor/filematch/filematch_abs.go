/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

type (
	AbsFileMatcher struct {
		Path string
	}
)

func (m *AbsFileMatcher) IsDynamicMultiFiles() bool {
	return false
}

func (m *AbsFileMatcher) Find() ([]FatPath, int, error) {
	return []FatPath{{Path: m.Path}}, 0, nil
}

func NewAbsFileMatcher(path string) *AbsFileMatcher {
	return &AbsFileMatcher{
		Path: path,
	}
}
