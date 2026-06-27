package models

type EventType int

const (
	EventTypeGenReport EventType = iota
	EventTypeDeleteSentReport
	EventTypeGenReportForTG
)

type Event struct {
	Name string
	Type EventType
}

func NewEvent(name string, t int) Event {
	var et EventType

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
