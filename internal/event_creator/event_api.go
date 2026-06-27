package eventcreator

import (
	"context"
	models "support_bot/internal/models/report"
)

type EventAPI struct {
	OutC   chan models.Event
	OutSpC chan models.SpecialEventForLK
}

func NewEventAPI(outC chan models.Event, outSpC chan models.SpecialEventForLK) *EventAPI {
	return &EventAPI{
		OutC:   outC,
		OutSpC: outSpC,
	}
}

func (api *EventAPI) ProduceScepialEvent(ctx context.Context, name string, recipient models.Recipient) {
	ev := models.SpecialEventForLK{
		Event: models.Event{
			Name: name,
			Type: models.EventTypeGenReportForTG,
		},
		Recipient: recipient,
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case api.OutSpC <- ev:
		}
	}()
}

func (api *EventAPI) ProduceGenEvent(ctx context.Context, name string) {
	ev := models.Event{
		Name: name,
		Type: models.EventTypeGenReport,
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case api.OutC <- ev:
		}
	}()
}

func (api *EventAPI) ProduceDelEvent(ctx context.Context, name string) {
	ev := models.Event{
		Name: name,
		Type: models.EventTypeDeleteSentReport,
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case api.OutC <- ev:
		}
	}()
}
