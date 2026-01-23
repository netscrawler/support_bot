package postgres

import (
	"net"
	"net/url"
	"strconv"
	"time"
)

const (
	sslDisable    = "disable"
	sslRequire    = "require"
	sslVerifyCa   = "verify-ca"
	sslVerifyFull = "verify-full"
	sslAllow      = "allow"
	sslPrefer     = "prefer"
)

type PostgresConfig struct {
	Port     int    `env:"DATABASE_PORT"     env-default:"5432"      yaml:"port"`
	Host     string `env:"DATABASE_HOST"     env-default:"localhost" yaml:"host"`
	User     string `env:"DATABASE_USER"     env-default:"user"      yaml:"user"`
	Password string `env:"DATABASE_PASSWORD" env-default:"password"  yaml:"password"`
	Name     string `env:"DATABASE_NAME"     env-default:"postgres"  yaml:"name"`
	SSL      string `env:"DATABASE_SSL_MODE" env-default:"false"     yaml:"sslmode"`

	MaxConns        int           `env:"DATABASE_MAX_CONNS"          env-default:"10"  yaml:"max_conns"`
	MaxIdleConns    int           `env:"DATABASE_MAX_IDLE_CONNS"     env-default:"2"   yaml:"max_idle_conns"`
	MaxConnLifeTime time.Duration `env:"DATABASE_MAX_CONN_LIFE_TIME" env-default:"30m" yaml:"max_conn_life_time"`
	MaxConnIdleTime time.Duration `env:"DATABASE_MAX_CONN_IDLE_TIME" env-default:"5m"  yaml:"max_conn_idle_time"`

	DatabaseConnect time.Duration `env:"DATABASE_CONNECT_TIMEOUT" env-default:"30s" yaml:"database_connect"`

	DSN string
}

func (pc *PostgresConfig) GetDSN() string {
	mode := sslDisable
	modes := map[string]struct{}{
		sslDisable:    {},
		sslRequire:    {},
		sslVerifyCa:   {},
		sslVerifyFull: {},
		sslAllow:      {},
		sslPrefer:     {},
	}

	if _, ok := modes[pc.SSL]; ok {
		mode = pc.SSL
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(pc.User, pc.Password),
		Host:   net.JoinHostPort(pc.Host, strconv.Itoa(pc.Port)),
		Path:   pc.Name,
	}

	q := dsn.Query()
	q.Set("sslmode", mode)
	q.Set("connect_timeout", strconv.Itoa(int(pc.DatabaseConnect.Seconds())))
	dsn.RawQuery = q.Encode()

	pc.DSN = dsn.String()

	return pc.DSN
}
