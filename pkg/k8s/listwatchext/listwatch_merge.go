package listwatchext

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/watch"
	"sync"
	"time"
)

type (
	BatchEventConsumer func(merged map[string]watch.Event)
	EventMerger        struct {
		merged   map[string]watch.Event
		mutex    sync.Mutex
		interval time.Duration
		consumer BatchEventConsumer
		stopCh   <-chan struct{}
	}
)

func NewEventMerger(interval time.Duration, consumer BatchEventConsumer, stopCh <-chan struct{}) *EventMerger {
	m := &EventMerger{
		interval: interval,
		merged:   make(map[string]watch.Event),
		consumer: consumer,
		stopCh:   stopCh,
	}
	m.start()
	return m
}

func (m *EventMerger) Add(e watch.Event) {
	if e.Type == watch.Error || e.Type == watch.Bookmark {
		return
	}

	mobj, err := meta.Accessor(e.Object)
	if err != nil {
		// TODO log ?
		return
	}
	var key string
	if len(mobj.GetNamespace()) > 0 {
		key = mobj.GetNamespace() + "/" + mobj.GetName()
	} else {
		key = mobj.GetName()
	}

	m.mutex.Lock()
	m.merged[key] = e
	m.mutex.Unlock()
}

func (m *EventMerger) start() {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.flush()
			}
		}
	}()
}

func (m *EventMerger) flush() {
	m.mutex.Lock()
	if len(m.merged) == 0 {
		m.mutex.Unlock()
		return
	}

	x := m.merged
	m.merged = make(map[string]watch.Event)
	m.mutex.Unlock()

	m.consumer(x)
}
