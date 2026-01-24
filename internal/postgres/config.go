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
	Port     int    `env:"DATABASE_PORT"     env-default:"5432"      yaml:"port"     comment:"Порт для подключения к базе"`
	Host     string `env:"DATABASE_HOST"     env-default:"localhost" yaml:"host"     comment:"Адрес базы данных"`
	User     string `env:"DATABASE_USER"     env-default:"user"      yaml:"user"                                                                               Comment:"Пользователь"`
	Password string `env:"DATABASE_PASSWORD" env-default:"password"  yaml:"password" comment:"Пароль для подключения"`
	Name     string `env:"DATABASE_NAME"     env-default:"postgres"  yaml:"name"     comment:"Имя базы данных"`
	SSL      string `env:"DATABASE_SSL_MODE" env-default:"false"     yaml:"sslmode"  comment:"SSL режим: (disable|require|verify-ca|verify-full|allow|prefer)"`

	MaxConns        int           `env:"DATABASE_MAX_CONNS"          env-default:"10"  yaml:"max_conns"          comment:"Максимально количество соединений"`
	MaxIdleConns    int           `env:"DATABASE_MAX_IDLE_CONNS"     env-default:"2"   yaml:"max_idle_conns"     comment:"Максимальное количество ожидающий соединений"`
	MaxConnLifeTime time.Duration `env:"DATABASE_MAX_CONN_LIFE_TIME" env-default:"30m" yaml:"max_conn_life_time" comment:"Максимальное время жизни одного соединения"`
	MaxConnIdleTime time.Duration `env:"DATABASE_MAX_CONN_IDLE_TIME" env-default:"5m"  yaml:"max_conn_idle_time" comment:"Максимальное время ожидания соединения"`

	DatabaseConnect time.Duration `env:"DATABASE_CONNECT_TIMEOUT" env-default:"30s" yaml:"database_connect" comment:"Таймаут на подключения к базе данных"`

	DSN string `yaml:"-"`
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
