package postgres

import (
	"fmt"
	"time"

	"support_bot/internal/pkg/secret"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DSN              secret.String `comment:"DSN подключения к Postgres"                                                    json:"DSN"              yaml:"dsn"`
	ReadOnly         *bool         `comment:"Только чтение (по умолчанию true)"                                             json:"ReadOnly"         yaml:"read_only"`
	StatementTimeout time.Duration `comment:"Таймаут запроса (по умолчанию 30s)"                                            json:"StatementTimeout" yaml:"statement_timeout"`
	MaxConns         int32         `comment:"Максимум соединений в пуле (по умолчанию 4)"                                   json:"MaxConns"         yaml:"max_conns"`
}

// withDefaults: теги env-default не работают для значений map, поэтому дефолты тут.
func (c Config) withDefaults() Config {
	if c.ReadOnly == nil {
		t := true

		c.ReadOnly = &t
	}

	if c.StatementTimeout == 0 {
		c.StatementTimeout = 30 * time.Second
	}

	if c.MaxConns == 0 {
		c.MaxConns = 4
	}

	return c
}

func poolConfig(cfg Config) (*pgxpool.Config, error) {
	cfg = cfg.withDefaults()

	pc, err := pgxpool.ParseConfig(cfg.DSN.Value())
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pc.MaxConns = cfg.MaxConns

	if *cfg.ReadOnly {
		pc.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	}

	return pc, nil
}
