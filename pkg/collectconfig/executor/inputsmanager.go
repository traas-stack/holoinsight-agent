/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

import (
	"github.com/traas-stack/holoinsight-agent/pkg/collectconfig/executor/filematch"
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
		FatPath filematch.FatPath
		Cursor  int64
		// lastState and state are used to detect state change
		LastState inputWrapperStatus
		State     inputWrapperStatus
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
		key := logstream.BuildFileKey(fatPath.Path, fatPath.Attrs)
		// deduplication
		if _, ok := usedPaths[key]; ok {
			continue
		}
		usedPaths[key] = struct{}{}

		if iw, ok := im.inputs[key]; ok {
			newInputs[key] = iw
		} else {
			// create if not exist
			logger.Infoz("[pipeline] [log] [input] add", //
				zap.String("key", im.key), //
				zap.String("path", key))

			var ls logstream.LogStream

			if fatPath.IsSls {
				ls = im.lsm.AcquireSls(fatPath.SlsConfig)
			} else {
				ls = im.lsm.AcquireFile(fatPath.Path, fatPath.Attrs)
			}
			newInputs[key] = &inputWrapper{
				ls: ls,
				inputStateObj: inputStateObj{
					FatPath:   fatPath,
					Cursor:    ls.AddListener(im.listener),
					State:     inputWrapperStateFirst,
					LastState: inputWrapperStateFirst,
				},
			}
		}
	}

	for key := range im.inputs {
		if _, ok := newInputs[key]; !ok {
			// 删除那些不再匹配的
			im.releaseStream(im.inputs[key])
			logger.Infoz("[pipeline] [log] [input] remove input", //
				zap.String("key", im.key), //
				zap.String("path", key),
				zap.Any("new", paths),
			)
		}
	}

	im.inputs = newInputs
}

func (im *inputsManager) releaseStream(iw *inputWrapper) {
	iw.ls.RemoveListener(im.listener, iw.Cursor)
	im.lsm.Release(iw.ls)
}

func (im *inputsManager) stop() {
	for k, iw := range im.inputs {
		im.releaseStream(iw)
		delete(im.inputs, k)
	}
}

func (im *inputsManager) update(st *api.SubTask) {
	im.ld = NewLogDetector(im.key, st.SqlTask.From, st.CT.Target)
	im.checkInputsChange()
}
