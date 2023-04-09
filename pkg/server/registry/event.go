package registry

import (
	"fmt"
	"github.com/traas-stack/holoinsight-agent/pkg/logger"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
	"go.uber.org/zap"
	"time"
)

const (
	// eventQueueSize is the chan size of event queue
	eventQueueSize = 4096
	// reportBatchWait is used to merge events which are generated very close in time
	reportBatchWait = 100 * time.Millisecond
)

type (
	eventManager struct {
		ch                 chan []*pb.ReportEventRequest_Event
		stopCh             chan struct{}
		eventBatchReporter eventBatchReporter
	}
	eventBatchReporter func(events []*pb.ReportEventRequest_Event) error
)

// newEventManager creates a new eventManager instance
func newEventManager(eventBatchReporter eventBatchReporter) *eventManager {
	return &eventManager{
		ch:                 make(chan []*pb.ReportEventRequest_Event, eventQueueSize),
		stopCh:             make(chan struct{}),
		eventBatchReporter: eventBatchReporter,
	}
}

// add adds events to queue
func (m *eventManager) add(events []*pb.ReportEventRequest_Event) {
	select {
	case m.ch <- events:
	default:
		m.discard(events, fmt.Errorf("event queue full %d", cap(m.ch)))
	}
}

// start starts the event manager background goroutine
func (m *eventManager) start() {
	go m.loop()
}

// stop stops the event manager background goroutine
func (m *eventManager) stop() {
	close(m.stopCh)
}

// loop consumes events in background
func (m *eventManager) loop() {
	for {
		select {
		case <-m.stopCh:
			return
		case events := <-m.ch:
			// reports events as many as possible
			time.Sleep(reportBatchWait)
			size := len(m.ch)
			for i := 0; i < size; i++ {
				events = append(events, <-m.ch...)
			}
			m.report(events)
		}
	}
}

// report reports batch events to server side
func (m *eventManager) report(events []*pb.ReportEventRequest_Event) {
	err := m.eventBatchReporter(events)
	if err != nil {
		m.discard(events, err)
	}
}

// discard is called when events are discarded
func (m *eventManager) discard(events []*pb.ReportEventRequest_Event, err error) {
	for _, event := range events {
		logger.Errorz("[registry] discard event", zap.Any("event", event), zap.Error(err))
	}
}
