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
	defer func() { recover() }()

	sha.c <- eventStart
}

func (sha *SheduleAPI) Stop() {
	defer func() { recover() }()

	sha.c <- eventStop
}

func (sha *SheduleAPI) restart() {
	defer func() { recover() }()

	sha.Stop()
	sha.Start()
}

func (sha *SheduleAPI) stopAPI() {
	close(sha.c)
}
