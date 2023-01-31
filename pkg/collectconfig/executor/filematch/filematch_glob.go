package filematch

import (
	"errors"
	"path/filepath"
)

type (
	GlobFileMatcher struct {
		pattern string
	}
)

func (m *GlobFileMatcher) IsDynamicMultiFiles() bool {
	return true
}

func NewGlobFileMatcher(pattern string) (*GlobFileMatcher, error) {
	if pattern == "" || pattern == "/" {
		return nil, errors.New("invalid pattern")
	}
	return &GlobFileMatcher{
		pattern: pattern,
	}, nil
}

func (m *GlobFileMatcher) Find() ([]FatPath, int, error) {
	// TODO 容器case能否直接在这里grep?
	files, err := filepath.Glob(m.pattern)
	ret := make([]FatPath, len(files))
	for i := range files {
		ret[i] = FatPath{Path: files[i]}
	}
	return ret, -1, err
}
