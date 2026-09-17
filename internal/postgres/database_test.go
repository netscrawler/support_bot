package postgres

import (
	"testing"
	"time"
)

func TestNew_PoolConfigMapping(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		Port:            5432,
		User:            "user",
		Password:        "password",
		Name:            "postgres",
		SSL:             "disable",
		MaxConns:        7,
		MaxIdleConns:    3,
		MaxConnLifeTime: 5 * time.Minute,
		MaxConnIdleTime: 2 * time.Minute,
	}

	pgxCfg, err := parsePoolConfig(cfg)
	if err != nil {
		t.Fatalf("parsePoolConfig() error = %v", err)
	}

	if pgxCfg.MaxConns != int32(cfg.MaxConns) { //nolint:gosec // test fixture value
		t.Errorf("MaxConns = %d, want %d", pgxCfg.MaxConns, cfg.MaxConns)
	}

	if pgxCfg.MaxConnLifetime != cfg.MaxConnLifeTime {
		t.Errorf("MaxConnLifetime = %v, want %v", pgxCfg.MaxConnLifetime, cfg.MaxConnLifeTime)
	}

	if pgxCfg.MaxConnIdleTime != cfg.MaxConnIdleTime {
		t.Errorf("MaxConnIdleTime = %v, want %v", pgxCfg.MaxConnIdleTime, cfg.MaxConnIdleTime)
	}
}
