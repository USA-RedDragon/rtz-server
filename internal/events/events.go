package events

type EventType string

const (
	EventTypeTest EventType = "test"
)

type Event interface {
	GetType() EventType
}

type TestEvent struct {
}

func (e TestEvent) GetType() EventType {
	return EventTypeTest
}

type EventBus struct {
	eventQueue chan Event
}

func NewEventBus() *EventBus {
	return &EventBus{
		eventQueue: make(chan Event, 100),
	}
}

func (eb *EventBus) GetChannel() chan Event {
	return eb.eventQueue
}
