package util

import (
	"runtime"
	"sync"
)

func GoWithRecover(handler func(), recoverHandlers ...func(p interface{})) {
	go WithRecover(handler, recoverHandlers...)
}

func WithRecover(handler func(), recoverHandlers ...func(p interface{})) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			//logger.Errorf("goroutine panic: %v\n%s", r, string(buf))

			for _, f := range recoverHandlers {
				if f != nil {
					f(r)
				}
			}
		}
	}()
	handler()
}

func GoWithSyncGroup(handler func(), wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		handler()
	}()
}
