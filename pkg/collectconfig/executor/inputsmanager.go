package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/pipeline/api"
	"go.uber.org/zap"
)

type (
	// inputWrapper wraps a logstream.ReadRequest
	inputWrapper struct {
		path     string
		pathTags map[string]string
		ls       logstream.LogStream
		req      *logstream.ReadRequest
		fileId   string
		// lastState and state are used to detect state change
		lastState inputWrapperState
		state     inputWrapperState
	}
	// inputsManager wraps inputs of a LogPipeline
	inputsManager struct {
		key      string
		ld       *LogPathDetector
		inputs   map[string]*inputWrapper
		listener *listenerImpl
		lsm      *logstream.Manager
	}
)

// check inputs change
func (im *inputsManager) checkInputsChange() {
	paths := im.ld.Detect()

	usedPaths := make(map[string]struct{})
	newInputs := make(map[string]*inputWrapper, len(paths))

	for _, fatPath := range paths {
		path := fatPath.Path
		// deduplication
		if _, ok := usedPaths[path]; ok {
			continue
		}
		usedPaths[path] = struct{}{}

		if iw, ok := im.inputs[path]; ok {
			newInputs[path] = iw
		} else {
			// create if not exist
			logger.Infoz("[pipeline] [log] [input] add", //
				zap.String("key", im.key), //
				zap.String("path", path))
			ls := im.lsm.Acquire(path)
			newInputs[path] = &inputWrapper{
				path:     path,
				pathTags: fatPath.Tags,
				ls:       ls,
				req: &logstream.ReadRequest{
					Cursor: ls.AddListener(im.listener),
				},
				state:     inputWrapperStateFirst,
				lastState: inputWrapperStateFirst,
			}
		}
	}

	for path := range im.inputs {
		if _, ok := newInputs[path]; !ok {
			// 删除那些不再匹配的
			im.releaseStream(im.inputs[path])
			logger.Infoz("[pipeline] [log] [input] input", //
				zap.String("key", im.key), //
				zap.String("path", path))
		}
	}

	im.inputs = newInputs
}

func (im *inputsManager) releaseStream(iw *inputWrapper) {
	iw.ls.RemoveListener(im.listener, iw.req.Cursor)
	im.lsm.Release(iw.path, iw.ls)
}

func (im *inputsManager) stop() {
	for k, iw := range im.inputs {
		im.releaseStream(iw)
		delete(im.inputs, k)
	}
}

func (im *inputsManager) update(st *api.SubTask) {
	im.ld = NewLogDetector(im.key, st.SqlTask.From.Log.Path, st.CT.Target)
	im.checkInputsChange()
}
