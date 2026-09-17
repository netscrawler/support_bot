package app

import (
	"context"
	"log/slog"
	"support_bot/internal/postgres"
	"support_bot/internal/reportdef"
	"support_bot/internal/store"

	"go.uber.org/fx"
)

var reportDefinitionModule = fx.Module("report_definition", fx.Provide(
	newReportDefinitionStore,
	newReportDefinitionCleaner,
))

func newReportDefinitionStore(rdb *postgres.DB, log *slog.Logger) *store.ReportDefinitionStore {
	return store.NewReportDefinitionStore(rdb.GetConn(), log)
}

func newReportDefinitionCleaner(
	appCtx context.Context,
	s *store.ReportDefinitionStore,
	log *slog.Logger,
	lc fx.Lifecycle,
) *reportdef.Cleaner {
	c := reportdef.NewCleaner(s, log)

	lc.Append(fx.Hook{OnStart: func(context.Context) error {
		c.Start(appCtx)

		return nil
	}})

	return c
}
