package eventcreator

import (
	"context"

	models2 "support_bot/internal/models"
)

type EventAPI struct {
	OutC   chan models2.Event
	OutSpC chan models2.SpecialEventForLK
}

func NewEventAPI(outC chan models2.Event, outSpC chan models2.SpecialEventForLK) *EventAPI {
	return &EventAPI{
		OutC:   outC,
		OutSpC: outSpC,
	}
}

func (api *EventAPI) ProduceSpecialEvent(
	ctx context.Context,
	name string,
	recipient models2.Recipient,
) {
	ev := models2.SpecialEventForLK{
		Event: models2.Event{
			Name: name,
			Type: models2.EventTypeGenReportForTG,
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

func (api *EventAPI) produceGenEvent(ctx context.Context, name string) {
	ev := models2.Event{
		Name: name,
		Type: models2.EventTypeGenReport,
	}

	go func() {
		select {
		case <-ctx.Done():
			return
		case api.OutC <- ev:
		}
	}()
}

func (api *EventAPI) produceDelEvent(ctx context.Context, name string) {
	ev := models2.Event{
		Name: name,
		Type: models2.EventTypeDeleteSentReport,
	}

	go func() {
		select {
		case <-ctx.Done():
			return
		case api.OutC <- ev:
		}
	}()
}
