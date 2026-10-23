package app

import (
	"context"
	"log/slog"
	apihttp "support_bot/internal/api/http"
	apihandlers "support_bot/internal/api/http/handlers"
	"support_bot/internal/config"
	"support_bot/internal/metrics"
	"support_bot/internal/postgres"
	reportsvc "support_bot/internal/service"

	"go.uber.org/fx"
)

var httpModule = fx.Module("http", fx.Provide(newReportHandler, newHTTPServer, newMetricsServer))

func newReportHandler(reportGenSvc *reportsvc.Report, log *slog.Logger) *apihandlers.Handler {
	return apihandlers.NewHandler(reportGenSvc, log)
}

func newHTTPServer(
	cfg *config.Config,
	reportHandler *apihandlers.Handler,
	log *slog.Logger,
	lc fx.Lifecycle,
) *apihttp.Server {
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

func newMetricsServer(
	cfg *config.Config,
	db *postgres.DB,
	log *slog.Logger,
	lc fx.Lifecycle,
) *metrics.Server {
	if !cfg.Metrics.Enabled {
		return nil
	}

	srv := metrics.NewServer(cfg.Metrics.Addr, db.GetConn().Ping, log)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			srv.Start()
			return nil
		},
		OnStop: srv.Shutdown,
	})

	return srv
}
