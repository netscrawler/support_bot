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
	defer recover()
	sha.c <- EventStart
}

func (sha *SheduleAPI) Stop() {
	defer recover()
	sha.c <- EventStop
}

func (sha *SheduleAPI) Restart() {
	defer recover()
	sha.Stop()
	sha.Start()
}

func (sha *SheduleAPI) StopAPI() {
	close(sha.c)
}
