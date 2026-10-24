package sheduler

type SheduleAPIEvent string

const (
	eventStart SheduleAPIEvent = "start"
	eventStop  SheduleAPIEvent = "stop"
)

type SheduleAPI struct {
	c chan SheduleAPIEvent
}

func NewSheduleAPI(c chan SheduleAPIEvent) *SheduleAPI {
	return &SheduleAPI{c: c}
}

func (sha *SheduleAPI) Start() {
	defer func() { recover() }() //nolint:errcheck // отправка в закрытый канал даёт панику, она ожидаема

	sha.c <- eventStart
}

func (sha *SheduleAPI) Stop() {
	defer func() { recover() }() //nolint:errcheck // отправка в закрытый канал даёт панику, она ожидаема

	sha.c <- eventStop
}
