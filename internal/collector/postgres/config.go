package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DSN              string        `yaml:"dsn"               comment:"DSN подключения к Postgres"`
	ReadOnly         *bool         `yaml:"read_only"         comment:"Только чтение (по умолчанию true)"`
	StatementTimeout time.Duration `yaml:"statement_timeout" comment:"Таймаут запроса (по умолчанию 30s)"`
	MaxConns         int32         `yaml:"max_conns"         comment:"Максимум соединений в пуле (по умолчанию 4)"`
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

	pc, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pc.MaxConns = cfg.MaxConns

	if *cfg.ReadOnly {
		pc.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	}

	return pc, nil
}
