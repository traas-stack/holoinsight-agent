/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package filematch

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
)

type (
	SlsFileMatch struct {
		SlsConfigs []logstream.SlsConfig
	}
)

func (s *SlsFileMatch) Find() ([]FatPath, int, error) {
	paths := make([]FatPath, len(s.SlsConfigs))
	for i := range s.SlsConfigs {
		paths[i] = FatPath{
			Path:      s.SlsConfigs[i].BuildKey(),
			IsSls:     true,
			SlsConfig: s.SlsConfigs[i],
		}
	}
	return paths, 0, nil
}

func (s *SlsFileMatch) IsDynamicMultiFiles() bool {
	return false
}
