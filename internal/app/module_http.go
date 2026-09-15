package app

import (
	"context"
	"log/slog"
	apihttp "support_bot/internal/api/http"
	apihandlers "support_bot/internal/api/http/handlers"
	"support_bot/internal/config"
	reportsvc "support_bot/internal/service"

	"go.uber.org/fx"
)

var httpModule = fx.Module("http", fx.Provide(newReportHandler, newHTTPServer))

func newReportHandler(reportGenSvc *reportsvc.Report, log *slog.Logger) *apihandlers.Handler {
	return apihandlers.NewHandler(reportGenSvc, log)
}

func newHTTPServer(cfg *config.Config, reportHandler *apihandlers.Handler, log *slog.Logger, lc fx.Lifecycle) *apihttp.Server {
	httpSrv := apihttp.New(&cfg.HTTP, reportHandler, log)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			httpSrv.Start()
			return nil
		},
		OnStop: httpSrv.Shutdown,
	})

	return httpSrv
}
