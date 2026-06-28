package models

type eventType int

const (
	EventTypeGenReport eventType = iota
	EventTypeDeleteSentReport
	EventTypeGenReportForTG
)

type Event struct {
	Name string
	Type eventType
}

func NewEvent(name string, t int) Event {
	var et eventType

	switch t {
	case int(EventTypeDeleteSentReport):
		et = EventTypeDeleteSentReport
	case int(EventTypeGenReport):
		et = EventTypeGenReport
	default:
		et = EventTypeGenReport
	}

	return Event{
		Name: name,
		Type: et,
	}
}

type SpecialEventForLK struct {
	Event
	Recipient
}
