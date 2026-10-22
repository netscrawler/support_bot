package app

import (
	"context"
	"log/slog"
	"support_bot/internal/collector"
	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/jira"
	"support_bot/internal/collector/metabase"
	"support_bot/internal/collector/postgres"
	"support_bot/internal/config"
	"support_bot/internal/pkg/retry"
	"time"

	"go.uber.org/fx"
)

const collectorsParallel uint8 = 30

var collectorsModule = fx.Module(
	"collectors",
	fx.Provide(
		newMetabase,
		newAppMetricaCollector,
		newJira,
		newPostgresCollector,
		newCollector,
		newRetry,
	),
)

func newMetabase(cfg *config.Config) *metabase.Metabase {
	return metabase.New(cfg.MetabaseDomain)
}

func newAppMetricaCollector(
	ctx context.Context,
	cfg *config.Config,
	log *slog.Logger,
) *appmetrica.Collector {
	appM := appmetrica.NewCollector(&cfg.AppMetrica, log)

	sup, err := appM.GetApplications(ctx)
	if err != nil {
		log.ErrorContext(
			ctx,
			"error getting available applications for app metrica collector",
			slog.Any("error", err),
		)
	} else {
		log.InfoContext(
			ctx,
			"get available apps for collect data from app metrica",
			slog.Any("apps", sup),
		)
	}

	return appM
}

func newJira(cfg *config.Config) *jira.Collector {
	return jira.New(cfg.Jira)
}

func newPostgresCollector(
	ctx context.Context,
	lc fx.Lifecycle,
	cfg *config.Config,
	log *slog.Logger,
) (*postgres.Collector, error) {
	pg, err := postgres.New(ctx, cfg.Databases, log)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{OnStop: func(context.Context) error {
		pg.Close()

		return nil
	}})

	return pg, nil
}

func newCollector(
	mb *metabase.Metabase,
	appM *appmetrica.Collector,
	jiraColl *jira.Collector,
	pg *postgres.Collector,
	log *slog.Logger,
) *collector.Collector {
	return collector.NewCollector(collectorsParallel, mb, appM, jiraColl, pg, log)
}

func newRetry(log *slog.Logger) *retry.Retry {
	return retry.New(retry.Config{
		QueueSize:  100,
		Workers:    4,
		MaxRetries: 3,
		Backoff: retry.ExponentialBackoff{
			Base: 3 * time.Second,
			Max:  30 * time.Second,
		},
		Policy: retry.PolicyAlways{},
		Logger: log,
		Silent: false,
	})
}
