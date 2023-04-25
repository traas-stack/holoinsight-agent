/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

type (
	RegexpFileMatcher struct {
		dir        string
		r          *regexp.Regexp
		maxVisited int
		maxMatched int
	}
)

func (m *RegexpFileMatcher) IsDynamicMultiFiles() bool {
	return true
}

func NewRegexpFileMatcher(dir, pattern string, maxVisited, maxMatched int) (*RegexpFileMatcher, error) {
	if dir == "" || dir == "/" {
		return nil, errors.New("invalid idr")
	}
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexpFileMatcher{
		dir:        dir,
		r:          r,
		maxVisited: maxVisited,
		maxMatched: maxMatched,
	}, nil
}

func (m *RegexpFileMatcher) Find() ([]FatPath, int, error) {
	return walkAndCollect(m.dir, m.r, m.maxVisited, m.maxMatched), 0, nil
}

func walkAndCollect(basedir string, r *regexp.Regexp, maxVisited, maxMatched int) []FatPath {
	// 匹配的文件: 不能太多, 否则要打开很多句柄
	matched := 0
	// 遍历涉及的文件, 不能太多, 否则每次都要遍历很多文件, 频率不能太高
	visited := 0
	var ret []FatPath
	filepath.WalkDir(basedir, func(path string, d fs.DirEntry, err error) error {
		if maxVisited > 0 {
			visited++
			if visited >= maxVisited {
				return filepath.SkipDir
			}
		}
		if err != nil {
			return nil
		}

		// skip hide files/dirs
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		sub := r.FindStringSubmatch(path)
		if len(sub) > 0 {
			var tags map[string]string
			if r.NumSubexp() > 0 {
				tags = make(map[string]string, r.NumSubexp())
				subNames := r.SubexpNames()
				for i := 1; i < len(subNames); i++ {
					tags[subNames[i]] = sub[i]
				}
			}
			ret = append(ret, FatPath{Path: path, Tags: tags})
			if maxMatched > 0 {
				matched++
				if matched >= maxMatched {
					return errors.New("stop")
				}
			}
		}

		return nil
	})
	return ret
}
