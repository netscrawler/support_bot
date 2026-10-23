package eventcreator

import (
	"context"
	"support_bot/internal/models"
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

func (api *EventAPI) ProduceSpecialEvent(
	ctx context.Context,
	name string,
	recipient models.Recipient,
) {
	api.produceSpecialEvent(ctx, name, recipient, false)
}

// ProduceTestEvent — то же, что ProduceSpecialEvent, но с признаком пробного запуска.
func (api *EventAPI) ProduceTestEvent(
	ctx context.Context,
	name string,
	recipient models.Recipient,
) {
	api.produceSpecialEvent(ctx, name, recipient, true)
}

func (api *EventAPI) ProduceGenEvent(ctx context.Context, name string) {
	ev := models.SpecialEventForLK{
		Event: models.Event{
			Name: name,
			Type: models.EventTypeGenReport,
		},
	}

	go func() {
		select {
		case <-ctx.Done():
			return
		case api.OutSpC <- ev:
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

func (api *EventAPI) produceSpecialEvent(
	ctx context.Context,
	name string,
	recipient models.Recipient,
	test bool,
) {
	ev := models.SpecialEventForLK{
		Event: models.Event{
			Name: name,
			Type: models.EventTypeGenReportForTG,
		},
		Recipient: recipient,
		Test:      test,
	}

	go func() {
		select {
		case <-ctx.Done():
			return
		case api.OutSpC <- ev:
		}
	}()
}
