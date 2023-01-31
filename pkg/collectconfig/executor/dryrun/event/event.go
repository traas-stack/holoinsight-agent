package event

import "fmt"

type (
	// Event is used to records structured events
	Event struct {
		Title    string                 `json:"title,omitempty"`
		Params   map[string]interface{} `json:"params,omitempty"`
		Messages []*Message             `json:"messages,omitempty"`
		Children []*Event               `json:"children,omitempty"`
	}
	// Message contains level and content
	Message struct {
		Level   string `json:"level,omitempty"`
		Content string `json:"content,omitempty"`
	}
	// WhereEvent is the event for 'where' execution.
	WhereEvent struct {
		// Name is the name of where op, such as 'and', 'contains'.
		Name string `json:"name,omitempty"`
		// Result is the bool result of where op
		Result bool `json:"result"`
		// Children contains sub where op events. For example, 'and' may contain 2 children WhereEvent.
		Children []*WhereEvent `json:"children,omitempty"`
	}
)

// AddChild adds a child to current event and returns new child
func (e *Event) AddChild(format string, args ...interface{}) *Event {
	child := &Event{}
	child.Title = fmt.Sprintf(format, args...)
	e.Children = append(e.Children, child)
	return child
}

// Set sets a property to current event
func (e *Event) Set(key string, value interface{}) *Event {
	if e.Params == nil {
		e.Params = make(map[string]interface{})
	}
	e.Params[key] = value
	return e
}

// Info logs with Info level
func (e *Event) Info(format string, args ...interface{}) *Event {
	return e.Log("INFO", format, args...)
}

// Error logs with Error level
func (e *Event) Error(format string, args ...interface{}) *Event {
	return e.Log("ERROR", format, args...)
}

// Log message
func (e *Event) Log(level string, format string, args ...interface{}) *Event {
	msg := fmt.Sprintf(format, args...)
	e.Messages = append(e.Messages, &Message{Level: level, Content: msg})
	return e
}

// AddChild adds a where child event and returns child
func (e *WhereEvent) AddChild() *WhereEvent {
	child := &WhereEvent{}
	e.Children = append(e.Children, child)
	return child
}
