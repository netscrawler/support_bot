package app

import (
	"context"
	"log/slog"

	"support_bot/internal/collector"
	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/grafana"
	"support_bot/internal/collector/httpcoll"
	"support_bot/internal/collector/jira"
	"support_bot/internal/collector/metabase"
	"support_bot/internal/collector/postgres"
	"support_bot/internal/collector/prometheus"
	"support_bot/internal/collector/zabbix"
	"support_bot/internal/config"
	"support_bot/internal/models"
	"support_bot/internal/pkg/secret"
	pgdb "support_bot/internal/postgres"
	"support_bot/internal/store"

	"go.uber.org/fx"
)

const (
	collectorsParallel uint8 = 30
)

var collectorsModule = fx.Module(
	"collectors",
	fx.Provide(
		newMetabase,
		newAppMetricaCollector,
		newJira,
		newPostgresCollector,
		newPrometheusCollector,
		newGrafanaAlertsCollector,
		newZabbixCollector,
		newCipher,
		newSecretStore,
		newHTTPCollector,
		newCollector,
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

func newPrometheusCollector(
	cfg *config.Config,
	log *slog.Logger,
) (*prometheus.Collector, error) {
	return prometheus.New(cfg.Prometheus, cfg.Grafana, log)
}

func newGrafanaAlertsCollector(
	cfg *config.Config,
	log *slog.Logger,
) *grafana.Collector {
	return grafana.New(cfg.Grafana, log)
}

func newZabbixCollector(
	cfg *config.Config,
	log *slog.Logger,
) (*zabbix.Collector, error) {
	return zabbix.New(cfg.Zabbix, log)
}

// newCipher возвращает nil без SECRETS_KEY: секреты из БД тогда недоступны.
func newCipher(cfg *config.Config) (*secret.Cipher, error) {
	if cfg.Secrets.Key == "" {
		return nil, nil //nolint:nilnil // ключ не задан — не ошибка
	}

	return secret.NewCipher(cfg.Secrets.Key.Value())
}

func newSecretStore(rdb *pgdb.DB, c *secret.Cipher) *store.SecretStore {
	return store.NewSecretStore(rdb.GetConn(), c)
}

func newHTTPCollector(
	cfg *config.Config,
	secrets *store.SecretStore,
	log *slog.Logger,
) (*httpcoll.Collector, error) {
	return httpcoll.New(cfg.HTTPSources, secrets, log)
}

func newCollector(
	mb *metabase.Metabase,
	appM *appmetrica.Collector,
	jiraColl *jira.Collector,
	pg *postgres.Collector,
	prom *prometheus.Collector,
	alerts *grafana.Collector,
	zbx *zabbix.Collector,
	httpColl *httpcoll.Collector,
	log *slog.Logger,
) *collector.Collector {
	return collector.NewCollector(collectorsParallel, collector.Fetchers{
		models.CollectTypeMetabase:      collector.IgnoreQuery(mb),
		models.CollectTypeAppMetrica:    collector.IgnoreQuery(appM),
		models.CollectTypeJIRA:          collector.IgnoreQuery(jiraColl),
		models.CollectTypePostgres:      pg,
		models.CollectTypePrometheus:    prom,
		models.CollectTypeGrafanaAlerts: alerts,
		models.CollectTypeZabbix:        zbx,
		models.CollectTypeHTTP:          httpColl,
	}, log)
}
