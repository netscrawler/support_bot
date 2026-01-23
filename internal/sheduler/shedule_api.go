package sheduler

type SheduleAPIEvent string

const (
	EventStart   SheduleAPIEvent = "start"
	EventStop    SheduleAPIEvent = "stop"
	EventRestart SheduleAPIEvent = "restart"
)

type SheduleAPI struct {
	c chan SheduleAPIEvent
}

func NewSheduleAPI(c chan SheduleAPIEvent) *SheduleAPI {
	return &SheduleAPI{c: c}
}

func (sha *SheduleAPI) Start() {
	defer func() { recover() }()

	sha.c <- EventStart
}

func (sha *SheduleAPI) Stop() {
	defer func() { recover() }()

	sha.c <- EventStop
}

func (sha *SheduleAPI) Restart() {
	defer func() { recover() }()

	sha.Stop()
	sha.Start()
}

func (sha *SheduleAPI) StopAPI() {
	close(sha.c)
}
