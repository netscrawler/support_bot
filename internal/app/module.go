package app

import (
	"go.uber.org/fx"

	apihttp "support_bot/internal/api/http"
	eventcreator "support_bot/internal/event_creator"
	"support_bot/internal/orchestrator"
	"support_bot/internal/sheduler"
	tgbot "support_bot/internal/tg_bot"
)

var Module = fx.Options(
	storageModule,
	collectorsModule,
	deliveryModule,
	telegramModule,
	reportPipelineModule,
	httpModule,
	fx.Invoke(func(*tgbot.Router, *apihttp.Server, *sheduler.Sheduler, *eventcreator.EventCreator, *orchestrator.Deleter) {
	}),
)
