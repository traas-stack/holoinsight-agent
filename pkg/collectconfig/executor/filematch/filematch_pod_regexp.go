/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"errors"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/cri"
	"github.com/traas-stack/holoinsight-agent/pkg/cri/criutils"
	"github.com/traas-stack/holoinsight-agent/pkg/ioc"
	"regexp"
	"strings"
)

type (
	PodRegexpFileMatcher struct {
		dir        string
		maxVisited int
		maxMatched int
		target     *collecttask.CollectTarget

		// 原始的正则表达式pattern
		originalPattern string

		// 上一次使用的正则表达式pattern
		lastPattern string
		lastRegexp  *regexp.Regexp
	}
)

func (m *PodRegexpFileMatcher) IsDynamicMultiFiles() bool {
	return true
}

func NewPodRegexpFileMatcher(target *collecttask.CollectTarget, dir, pattern string, maxVisited, maxMatched int) (*PodRegexpFileMatcher, error) {
	if dir == "" || dir == "/" {
		return nil, errors.New("invalid idr")
	}
	return &PodRegexpFileMatcher{
		target:          target,
		dir:             dir,
		maxVisited:      maxVisited,
		maxMatched:      maxMatched,
		originalPattern: pattern,
	}, nil
}

func (m *PodRegexpFileMatcher) Find() ([]FatPath, int, error) {

	container, err := criutils.GetMainBizContainerE(ioc.Crii, m.target.GetNamespace(), m.target.GetPodName())
	if err != nil {
		return nil, 0, err
	}

	// 方法1:
	// 我们认为被采集的文件一定在 volumes 下, 那么可以直接transfer父目录到物理机目录, 然后替换一下结果重新match
	basedirInHost, err := cri.TransferToHostPath0(container, m.dir, true)
	if err != nil {
		return nil, 0, err
	}

	replacedPattern := strings.Replace(m.originalPattern, m.dir, basedirInHost, 1)

	var r *regexp.Regexp
	if m.lastPattern == replacedPattern {
		r = m.lastRegexp
	} else {
		r, err = regexp.Compile(replacedPattern)
		if err != nil {
			return nil, 0, err
		}
		m.lastRegexp = r
	}

	if r == nil {
		return nil, 0, errors.New("invalid regexp")
	}

	return walkAndCollect(basedirInHost, r, m.maxVisited, m.maxMatched), 0, nil

	// 方法2:
	// 在容器内部执行(一次cmd执行) regexp, 获得匹配结果, 然后转成物理机上的地址
}
