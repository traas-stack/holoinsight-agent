package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig"
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/filematch"
	"github.com/traas-stack/holoinsight-agent/pkg/collecttask"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"go.uber.org/zap"
)

type (
	// 检测匹配哪些路径
	// TODO format 要特殊处理下, 它和其他的不太一样
	// 其他的都是lazy感应的, 而format需要实时感应
	LogPathDetector struct {
		key      string
		matchers []filematch.FileMatcher
	}
)

// TODO 这东西会变化...
// 如果当时pod不可用 那么这个方法会失败从而忽略它的路径
// 如果后来pod变得可用了, 由于该方法已经执行过, 因此不会再重试...
func NewLogDetector(key string, paths []*collectconfig.FromLogPath, target *collecttask.CollectTarget) *LogPathDetector {
	// TODO 考虑 daemonset case

	matchers := make([]filematch.FileMatcher, 0, len(paths))
	for _, path := range paths {
		switch path.Type {
		case filematch.TypePath:
			if target.IsTypePod() {
				matchers = append(matchers, filematch.NewPodAbsFileMatcher(target, path.Pattern))
			} else {
				matchers = append(matchers, filematch.NewAbsFileMatcher(path.Pattern))
			}
		case filematch.TypeGlob:
			m, err := filematch.NewGlobFileMatcher(path.Pattern)
			if err != nil {
				logger.Errorz("NewRegexpFileMatcher error", zap.Error(err))
				continue
			}
			matchers = append(matchers, m)
		case filematch.TypeRegexp:

			var m filematch.FileMatcher
			var err error
			if target.IsTypePod() {
				m, err = filematch.NewPodRegexpFileMatcher(target, path.Dir, path.Pattern, 1000, 1000)
			} else {
				m, err = filematch.NewRegexpFileMatcher(path.Dir, path.Pattern, 1000, 1000)
			}
			if err != nil {
				logger.Errorz("NewRegexpFileMatcher error", zap.Error(err))
				continue
			}
			matchers = append(matchers, m)
		case filematch.TypeFormat:
			matchers = append(matchers, filematch.NewFormatFileMatcher(path.Pattern))
		}
	}
	return &LogPathDetector{
		key:      key,
		matchers: matchers,
	}
}

func (ld *LogPathDetector) Detect() []filematch.FatPath {
	var newPaths []filematch.FatPath

	for _, m := range ld.matchers {
		paths, _, err := m.Find()
		if err != nil {
			logger.Errorz("[LogPathDetector] error", zap.String("key", ld.key), zap.Error(err))
			continue
		}
		newPaths = append(newPaths, paths...)
	}

	return newPaths
}
