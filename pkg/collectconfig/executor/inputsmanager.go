/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/logstream"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/plugin/api"
	"go.uber.org/zap"
)

type (
	// inputWrapper wraps a logstream.ReadRequest
	inputWrapper struct {
		ls logstream.LogStream
		inputStateObj
	}
	inputStateObj struct {
		Path     string
		PathTags map[string]string
		Cursor   int64
		// lastState and state are used to detect state change
		LastState inputWrapperStatus
		State     inputWrapperStatus
		FileId    string
		Offset    int64
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
				ls: ls,
				inputStateObj: inputStateObj{
					Path:      path,
					PathTags:  fatPath.Tags,
					Cursor:    ls.AddListener(im.listener),
					State:     inputWrapperStateFirst,
					LastState: inputWrapperStateFirst,
				},
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
	iw.ls.RemoveListener(im.listener, iw.Cursor)
	im.lsm.Release(iw.Path, iw.ls)
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
