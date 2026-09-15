package app

import "go.uber.org/fx"

var Module = fx.Options(
	storageModule,
	collectorsModule,
	deliveryModule,
	telegramModule,
	reportPipelineModule,
	httpModule,
)
