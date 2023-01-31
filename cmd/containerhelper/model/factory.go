package model

import "sync"

var mutex sync.Mutex
var handlers = make(map[string]Handler)

func RegisterHandler(inputType string, handler Handler) {
	mutex.Lock()
	defer mutex.Unlock()
	handlers[inputType] = handler
}

func GetHandler(inputType string) (Handler, bool) {
	mutex.Lock()
	defer mutex.Unlock()
	h, ok := handlers[inputType]
	return h, ok
}
