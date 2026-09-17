package app

import (
	"context"
	"support_bot/internal/orchestrator"
	"support_bot/internal/sheduler"

	"go.uber.org/fx"

	apihttp "support_bot/internal/api/http"
	eventcreator "support_bot/internal/event_creator"

	tgbot "support_bot/internal/tg_bot"
)

var Module = fx.Options(
	storageModule,
	collectorsModule,
	deliveryModule,
	telegramModule,
	reportPipelineModule,
	httpModule,
	// fx's dependency graph is lazy — these components have no other
	// dependents, so force their construction here to register their
	// fx.Lifecycle hooks.
	fx.Invoke(func(
		*tgbot.Router,
		*apihttp.Server,
		*sheduler.Sheduler,
		*eventcreator.EventCreator,
		*orchestrator.Deleter,
	) {
	}),
	// Registered after the terminal-component Invoke above, so this hook is
	// appended to the fx.Lifecycle last. fx runs OnStop hooks in reverse
	// registration order, so cancel() — which stops every appCtx-scoped
	// goroutine (sheduler, event creator, deleter, generator, orchestrator)
	// — fires FIRST on shutdown, before rdb.Stop() and the other OnStop
	// hooks registered earlier in the graph.
	fx.Invoke(func(lc fx.Lifecycle, cancel context.CancelFunc) {
		lc.Append(fx.Hook{OnStop: func(context.Context) error {
			cancel()
			return nil
		}})
	}),
)
